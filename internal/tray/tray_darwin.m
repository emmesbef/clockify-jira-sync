#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>

#include "tray_darwin.h"

// Go callback declarations (defined in tray_darwin.go via //export)
extern void goTrayShow(void);
extern void goTrayQuit(void);
extern void goTrayCheckUpdates(void);
extern void goTrayStartTimer(char *ticketKey, char *description);
extern void goTrayStopTimer(void);
extern char *goTrayLoadAssignedTickets(void);
extern char *goTraySearchTickets(char *query);

static NSStatusItem *statusItem = nil;
static NSMenuItem *showHideItem = nil;
static NSMenuItem *timerActionItem = nil;
static BOOL windowVisible = YES;
static BOOL trayTimerRunning = NO;
static NSImage *trayIconImage = nil;
static NSString *trayStatusText = @"";
static NSString *appVersion = nil;

@interface TrayStartController : NSObject <NSTableViewDataSource, NSTableViewDelegate, NSSearchFieldDelegate>
@property (nonatomic, strong) NSPopover *popover;
@property (nonatomic, strong) NSSearchField *descriptionField;
@property (nonatomic, strong) NSTableView *tableView;
@property (nonatomic, strong) NSArray<NSDictionary *> *tickets;
@property (nonatomic, strong) NSDictionary *selectedTicket;
@property (nonatomic, assign) BOOL updatingFromSelection;
@end

@interface TrayDelegate : NSObject
- (void)showWindow:(id)sender;
- (void)timerAction:(id)sender;
- (void)showAbout:(id)sender;
- (void)checkUpdates:(id)sender;
- (void)quitApp:(id)sender;
@end

static TrayDelegate *trayDelegate = nil;
static TrayStartController *trayStartController = nil;

static NSString *trimmedString(NSString *value) {
    if (!value) {
        return @"";
    }
    return [value stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceAndNewlineCharacterSet]];
}

static NSString *extractTicketKey(NSString *input) {
    NSString *candidate = [input uppercaseString];
    if ([candidate length] == 0) {
        return @"";
    }

    NSError *regexError = nil;
    NSRegularExpression *regex = [NSRegularExpression regularExpressionWithPattern:@"[A-Z][A-Z0-9]+-\\d+"
                                                                            options:0
                                                                              error:&regexError];
    if (regexError || !regex) {
        return @"";
    }

    NSRange range = [regex rangeOfFirstMatchInString:candidate options:0 range:NSMakeRange(0, [candidate length])];
    if (range.location == NSNotFound) {
        return @"";
    }

    return [candidate substringWithRange:range];
}

static NSArray<NSDictionary *> *ticketsFromGoJSON(char *jsonCString) {
    if (jsonCString == NULL) {
        return @[];
    }

    NSString *jsonString = [NSString stringWithUTF8String:jsonCString];
    free(jsonCString);
    if (!jsonString || [jsonString length] == 0) {
        return @[];
    }

    NSData *data = [jsonString dataUsingEncoding:NSUTF8StringEncoding];
    if (!data) {
        return @[];
    }

    NSError *parseError = nil;
    id parsed = [NSJSONSerialization JSONObjectWithData:data options:0 error:&parseError];
    if (parseError || ![parsed isKindOfClass:[NSArray class]]) {
        return @[];
    }

    NSMutableArray<NSDictionary *> *normalized = [NSMutableArray array];
    for (id item in (NSArray *)parsed) {
        if (![item isKindOfClass:[NSDictionary class]]) {
            continue;
        }

        NSDictionary *dict = (NSDictionary *)item;
        NSString *key = [dict[@"key"] isKindOfClass:[NSString class]] ? dict[@"key"] : @"";
        if ([key length] == 0) {
            continue;
        }

        NSString *summary = [dict[@"summary"] isKindOfClass:[NSString class]] ? dict[@"summary"] : @"";
        [normalized addObject:@{
            @"key": key,
            @"summary": summary,
        }];
    }

    return normalized;
}

@implementation TrayStartController

- (instancetype)init {
    self = [super init];
    if (self) {
        [self buildUI];
        self.tickets = @[];
    }
    return self;
}

- (void)buildUI {
    NSViewController *contentController = [[NSViewController alloc] init];
    NSView *rootView = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 420, 280)];
    contentController.view = rootView;

    NSSearchField *descriptionField = [[NSSearchField alloc] initWithFrame:NSMakeRect(16, 236, 388, 28)];
    [descriptionField setPlaceholderString:@"Description (e.g. PROJ-123 Work item)"];
    [descriptionField setDelegate:self];
    [rootView addSubview:descriptionField];
    self.descriptionField = descriptionField;

    NSScrollView *scrollView = [[NSScrollView alloc] initWithFrame:NSMakeRect(16, 62, 388, 164)];
    [scrollView setHasVerticalScroller:YES];

    NSTableView *tableView = [[NSTableView alloc] initWithFrame:scrollView.bounds];
    NSTableColumn *column = [[NSTableColumn alloc] initWithIdentifier:@"ticket"];
    [column setWidth:388];
    [tableView addTableColumn:column];
    [tableView setHeaderView:nil];
    [tableView setDataSource:self];
    [tableView setDelegate:self];
    [tableView setRowHeight:32];
    [tableView setTarget:self];
    [tableView setDoubleAction:@selector(startPressed:)];

    [scrollView setDocumentView:tableView];
    [rootView addSubview:scrollView];
    self.tableView = tableView;

    NSButton *cancelButton = [NSButton buttonWithTitle:@"Cancel" target:self action:@selector(cancelPressed:)];
    [cancelButton setFrame:NSMakeRect(236, 16, 80, 32)];
    [rootView addSubview:cancelButton];

    NSButton *startButton = [NSButton buttonWithTitle:@"Start" target:self action:@selector(startPressed:)];
    [startButton setFrame:NSMakeRect(324, 16, 80, 32)];
    [rootView addSubview:startButton];

    NSPopover *popover = [[NSPopover alloc] init];
    popover.contentViewController = contentController;
    popover.contentSize = NSMakeSize(420, 280);
    popover.behavior = NSPopoverBehaviorTransient;
    popover.animates = YES;
    self.popover = popover;
}

- (void)showFromStatusButton:(NSStatusBarButton *)button {
    if (!button || !self.popover) {
        return;
    }

    if (self.popover.isShown) {
        [self.popover close];
        return;
    }

    [NSObject cancelPreviousPerformRequestsWithTarget:self selector:@selector(runDebouncedSearch) object:nil];

    self.updatingFromSelection = YES;
    self.descriptionField.stringValue = @"";
    self.updatingFromSelection = NO;
    self.selectedTicket = nil;
    [self loadAssignedTickets];
    [self.tableView deselectAll:nil];

    [self.popover showRelativeToRect:button.bounds ofView:button preferredEdge:NSRectEdgeMinY];
    dispatch_async(dispatch_get_main_queue(), ^{
        [self.descriptionField.window makeFirstResponder:self.descriptionField];
    });
}

- (void)cancelPressed:(id)sender {
    [self.popover close];
}

- (void)startPressed:(id)sender {
    NSString *description = trimmedString(self.descriptionField.stringValue);
    if ([description length] == 0) {
        NSBeep();
        return;
    }

    NSString *ticketKey = @"";
    if ([self.selectedTicket isKindOfClass:[NSDictionary class]]) {
        NSString *candidate = self.selectedTicket[@"key"];
        if ([candidate isKindOfClass:[NSString class]]) {
            ticketKey = trimmedString(candidate);
        }
    }
    if ([ticketKey length] == 0) {
        ticketKey = extractTicketKey(description);
    }
    if ([ticketKey length] == 0) {
        NSBeep();
        return;
    }

    goTrayStartTimer((char *)[ticketKey UTF8String], (char *)[description UTF8String]);
    [self.popover close];
}

- (void)loadAssignedTickets {
    self.tickets = ticketsFromGoJSON(goTrayLoadAssignedTickets());
    [self.tableView reloadData];
}

- (void)runDebouncedSearch {
    NSString *query = trimmedString(self.descriptionField.stringValue);
    if ([query length] == 0) {
        [self loadAssignedTickets];
        return;
    }

    const char *utf8 = [query UTF8String];
    char *queryCopy = utf8 ? strdup(utf8) : strdup("");
    char *jsonResponse = goTraySearchTickets(queryCopy);
    free(queryCopy);

    self.tickets = ticketsFromGoJSON(jsonResponse);
    [self.tableView reloadData];
}

- (void)controlTextDidChange:(NSNotification *)obj {
    if (!self.updatingFromSelection) {
        self.selectedTicket = nil;
    }

    [NSObject cancelPreviousPerformRequestsWithTarget:self selector:@selector(runDebouncedSearch) object:nil];
    NSString *query = trimmedString(self.descriptionField.stringValue);
    if ([query length] == 0) {
        [self loadAssignedTickets];
        return;
    }

    [self performSelector:@selector(runDebouncedSearch) withObject:nil afterDelay:0.25];
}

- (NSInteger)numberOfRowsInTableView:(NSTableView *)tableView {
    return self.tickets.count;
}

- (NSView *)tableView:(NSTableView *)tableView viewForTableColumn:(NSTableColumn *)tableColumn row:(NSInteger)row {
    static NSString *identifier = @"TrayTicketCell";
    NSTableCellView *cell = [tableView makeViewWithIdentifier:identifier owner:self];
    if (!cell) {
        cell = [[NSTableCellView alloc] initWithFrame:NSMakeRect(0, 0, tableColumn.width, 32)];
        cell.identifier = identifier;

        NSTextField *label = [[NSTextField alloc] initWithFrame:NSMakeRect(6, 6, tableColumn.width - 12, 20)];
        [label setEditable:NO];
        [label setBezeled:NO];
        [label setDrawsBackground:NO];
        [label setSelectable:NO];
        [label setLineBreakMode:NSLineBreakByTruncatingTail];
        [cell addSubview:label];
        cell.textField = label;
    }

    NSDictionary *ticket = self.tickets[row];
    NSString *key = [ticket[@"key"] isKindOfClass:[NSString class]] ? ticket[@"key"] : @"";
    NSString *summary = [ticket[@"summary"] isKindOfClass:[NSString class]] ? ticket[@"summary"] : @"";
    if ([summary length] > 0) {
        cell.textField.stringValue = [NSString stringWithFormat:@"%@  %@", key, summary];
    } else {
        cell.textField.stringValue = key;
    }

    return cell;
}

- (void)tableViewSelectionDidChange:(NSNotification *)notification {
    NSInteger row = self.tableView.selectedRow;
    if (row < 0 || row >= (NSInteger)self.tickets.count) {
        self.selectedTicket = nil;
        return;
    }

    NSDictionary *ticket = self.tickets[(NSUInteger)row];
    self.selectedTicket = ticket;

    NSString *key = [ticket[@"key"] isKindOfClass:[NSString class]] ? ticket[@"key"] : @"";
    NSString *summary = [ticket[@"summary"] isKindOfClass:[NSString class]] ? ticket[@"summary"] : @"";
    NSString *value = [summary length] > 0 ? [NSString stringWithFormat:@"%@ %@", key, summary] : key;

    self.updatingFromSelection = YES;
    self.descriptionField.stringValue = value;
    self.updatingFromSelection = NO;
}

@end

@implementation TrayDelegate

- (void)showWindow:(id)sender {
    goTrayShow();
}

- (void)timerAction:(id)sender {
    if (trayTimerRunning) {
        goTrayStopTimer();
        return;
    }

    if (!statusItem || !statusItem.button || !trayStartController) {
        return;
    }
    dispatch_async(dispatch_get_main_queue(), ^{
        [trayStartController showFromStatusButton:statusItem.button];
    });
}

- (void)showAbout:(id)sender {
    NSAlert *alert = [[NSAlert alloc] init];
    [alert setMessageText:@"JiraFy Clockwork"];
    [alert setInformativeText:[NSString stringWithFormat:@"Version %@\n\nDesktop app to sync Clockify time entries with Jira worklogs.",
                               appVersion ?: @"dev"]];
    [alert setAlertStyle:NSAlertStyleInformational];
    [alert addButtonWithTitle:@"OK"];
    [alert runModal];
}

- (void)checkUpdates:(id)sender {
    goTrayCheckUpdates();
}

- (void)quitApp:(id)sender {
    goTrayQuit();
}

@end

void updateShowHideTitle(void) {
    if (showHideItem) {
        [showHideItem setTitle:windowVisible ? @"Hide Window" : @"Show Window"];
    }
}

void updateTimerActionTitle(void) {
    if (!timerActionItem) {
        return;
    }
    [timerActionItem setTitle:trayTimerRunning ? @"Stop Timer" : @"Start Timer…"];
}

void updateStatusButton(void) {
    if (!statusItem || !statusItem.button) {
        return;
    }

    NSString *status = trayStatusText ?: @"";
    BOOL hasStatus = [status length] > 0;
    BOOL hasIcon = trayIconImage != nil;

    statusItem.button.image = trayIconImage;

    if (hasStatus) {
        statusItem.length = NSVariableStatusItemLength;
        if (hasIcon) {
            statusItem.button.title = [NSString stringWithFormat:@" %@", status];
        } else {
            statusItem.button.title = [NSString stringWithFormat:@"⏱ %@", status];
        }
        return;
    }

    if (hasIcon) {
        statusItem.length = NSSquareStatusItemLength;
        statusItem.button.title = @"";
    } else {
        statusItem.length = NSVariableStatusItemLength;
        statusItem.button.title = @"⏱";
    }
}

void initTray(const char *version, const void *iconData, int iconLen) {
    // Copy data before dispatch_async — the Go caller frees the C string and
    // the icon pointer may become invalid after Init() returns.
    NSString *versionCopy = version ? [NSString stringWithUTF8String:version] : @"dev";
    NSData *iconDataCopy = (iconData && iconLen > 0)
        ? [NSData dataWithBytes:iconData length:iconLen]
        : nil;

    dispatch_async(dispatch_get_main_queue(), ^{
        appVersion = versionCopy;
        trayDelegate = [[TrayDelegate alloc] init];
        trayStartController = [[TrayStartController alloc] init];

        statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];

        if (iconDataCopy) {
            trayIconImage = [[NSImage alloc] initWithData:iconDataCopy];
            [trayIconImage setSize:NSMakeSize(18, 18)];
            trayIconImage.template = YES;
        }
        updateStatusButton();

        statusItem.button.toolTip = @"JiraFy Clockwork";

        NSMenu *menu = [[NSMenu alloc] init];

        showHideItem = [[NSMenuItem alloc] initWithTitle:@"Hide Window"
                                                  action:@selector(showWindow:)
                                           keyEquivalent:@""];
        [showHideItem setTarget:trayDelegate];
        [menu addItem:showHideItem];

        timerActionItem = [[NSMenuItem alloc] initWithTitle:@"Start Timer…"
                                                     action:@selector(timerAction:)
                                              keyEquivalent:@""];
        [timerActionItem setTarget:trayDelegate];
        [menu addItem:timerActionItem];
        updateTimerActionTitle();

        [menu addItem:[NSMenuItem separatorItem]];

        NSString *aboutTitle = [NSString stringWithFormat:@"About (v%@)", appVersion];
        NSMenuItem *aboutItem = [[NSMenuItem alloc] initWithTitle:aboutTitle
                                                           action:@selector(showAbout:)
                                                    keyEquivalent:@""];
        [aboutItem setTarget:trayDelegate];
        [menu addItem:aboutItem];

        NSMenuItem *updateItem = [[NSMenuItem alloc] initWithTitle:@"Check for Updates…"
                                                            action:@selector(checkUpdates:)
                                                     keyEquivalent:@""];
        [updateItem setTarget:trayDelegate];
        [menu addItem:updateItem];

        [menu addItem:[NSMenuItem separatorItem]];

        NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit"
                                                          action:@selector(quitApp:)
                                                   keyEquivalent:@"q"];
        [quitItem setTarget:trayDelegate];
        [menu addItem:quitItem];

        statusItem.menu = menu;

    });
}

void setTrayWindowVisible(int visible) {
    windowVisible = visible ? YES : NO;
    dispatch_async(dispatch_get_main_queue(), ^{
        updateShowHideTitle();
    });
}

void setTrayStatusText(const char *text) {
    NSString *textCopy = (text && text[0] != '\0') ? [NSString stringWithUTF8String:text] : @"";
    dispatch_async(dispatch_get_main_queue(), ^{
        trayStatusText = textCopy ?: @"";
        updateStatusButton();
    });
}

void setTrayTimerRunning(int running) {
    trayTimerRunning = running ? YES : NO;
    dispatch_async(dispatch_get_main_queue(), ^{
        updateTimerActionTitle();
    });
}

void setTrayAppBackgroundMode(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        BOOL applied = [NSApp setActivationPolicy:NSApplicationActivationPolicyProhibited];
        if (!applied) {
            [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
        }
        [NSApp hide:nil];
        [NSApp deactivate];
        dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(200 * NSEC_PER_MSEC)),
                       dispatch_get_main_queue(), ^{
            BOOL reapplied = [NSApp setActivationPolicy:NSApplicationActivationPolicyProhibited];
            if (!reapplied) {
                [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
            }
            [NSApp hide:nil];
            [NSApp deactivate];
        });
    });
}

void setTrayAppForegroundMode(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
        [NSApp unhide:nil];
        [NSApp activateIgnoringOtherApps:YES];
    });
}
