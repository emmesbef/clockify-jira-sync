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
extern void goTrayCancelTimer(void);
extern char *goTrayLoadAssignedTickets(void);
extern char *goTraySearchTickets(char *query);

static NSStatusItem *statusItem = nil;
static NSMenuItem *showHideItem = nil;
static NSMenuItem *timerActionItem = nil;
static NSMenuItem *cancelTimerItem = nil;
static NSMenuItem *statusDetailItem = nil;
static NSView *statusDetailView = nil;
static NSTextField *statusDetailLabel = nil;
static NSPopover *statusHoverPopover = nil;
static NSView *statusHoverView = nil;
static NSTextField *statusHoverLabel = nil;
static BOOL windowVisible = YES;
static BOOL trayTimerRunning = NO;
static NSImage *trayIconImage = nil;
static NSString *trayStatusText = @"";
static NSString *trayDetailText = @"";
static NSString *lastHoverLayoutText = @"";
static NSString *appVersion = nil;

void showStatusHoverPopover(void);
void hideStatusHoverPopover(void);
void updateStatusHoverPopoverContent(NSString *detailText);

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
- (void)cancelTimer:(id)sender;
- (void)mouseEntered:(NSEvent *)event;
- (void)mouseExited:(NSEvent *)event;
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
    [tableView setRowHeight:40];
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

- (CGFloat)tableView:(NSTableView *)tableView heightOfRow:(NSInteger)row {
    if (row < 0 || row >= (NSInteger)self.tickets.count) {
        return 40.0;
    }

    NSDictionary *ticket = self.tickets[(NSUInteger)row];
    NSString *key = [ticket[@"key"] isKindOfClass:[NSString class]] ? ticket[@"key"] : @"";
    NSString *summary = [ticket[@"summary"] isKindOfClass:[NSString class]] ? ticket[@"summary"] : @"";
    NSString *displayValue = [summary length] > 0 ? [NSString stringWithFormat:@"%@  %@", key, summary] : key;

    CGFloat textWidth = MAX(tableView.bounds.size.width - 12.0, 120.0);
    NSRect textRect = [displayValue boundingRectWithSize:NSMakeSize(textWidth, CGFLOAT_MAX)
                                                 options:(NSStringDrawingUsesLineFragmentOrigin | NSStringDrawingUsesFontLeading)
                                              attributes:@{NSFontAttributeName: [NSFont systemFontOfSize:[NSFont systemFontSize]]}];

    CGFloat rowHeight = ceil(textRect.size.height) + 8.0;
    return MAX(rowHeight, 28.0);
}

- (NSView *)tableView:(NSTableView *)tableView viewForTableColumn:(NSTableColumn *)tableColumn row:(NSInteger)row {
    static NSString *identifier = @"TrayTicketCell";
    NSTableCellView *cell = [tableView makeViewWithIdentifier:identifier owner:self];
    if (!cell) {
        cell = [[NSTableCellView alloc] initWithFrame:NSMakeRect(0, 0, tableColumn.width, 40)];
        cell.identifier = identifier;

        NSTextField *label = [[NSTextField alloc] initWithFrame:NSMakeRect(6, 4, tableColumn.width - 12, 32)];
        [label setEditable:NO];
        [label setBezeled:NO];
        [label setDrawsBackground:NO];
        [label setSelectable:NO];
        [label setLineBreakMode:NSLineBreakByWordWrapping];
        [label setUsesSingleLineMode:NO];
        [label setMaximumNumberOfLines:0];
        [cell addSubview:label];
        cell.textField = label;
    }

    NSDictionary *ticket = self.tickets[row];
    NSString *key = [ticket[@"key"] isKindOfClass:[NSString class]] ? ticket[@"key"] : @"";
    NSString *summary = [ticket[@"summary"] isKindOfClass:[NSString class]] ? ticket[@"summary"] : @"";
    NSString *displayValue = [summary length] > 0 ? [NSString stringWithFormat:@"%@  %@", key, summary] : key;
    CGFloat rowHeight = [tableView rectOfRow:row].size.height;
    if (rowHeight <= 0) {
        rowHeight = 40;
    }
    cell.frame = NSMakeRect(0, 0, tableColumn.width, rowHeight);
    cell.textField.frame = NSMakeRect(6, 4, tableColumn.width - 12, MAX(rowHeight - 8, 20));
    cell.textField.stringValue = displayValue;
    cell.toolTip = displayValue;

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
    hideStatusHoverPopover();
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

- (void)cancelTimer:(id)sender {
    goTrayCancelTimer();
}

- (void)mouseEntered:(NSEvent *)event {
    (void)event;
}

- (void)mouseExited:(NSEvent *)event {
    (void)event;
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
    if (cancelTimerItem) {
        cancelTimerItem.hidden = !trayTimerRunning;
    }
}

void hideStatusHoverPopover(void) {
    if (statusHoverPopover && statusHoverPopover.isShown) {
        [statusHoverPopover close];
    }
}

void updateStatusHoverPopoverContent(NSString *detailText) {
    if (!statusHoverPopover || !statusHoverView || !statusHoverLabel) {
        return;
    }

    NSString *trimmed = trimmedString(detailText ?: @"");
    if ([trimmed length] == 0 || [trimmed isEqualToString:@"JiraFy Clockwork"]) {
        statusHoverLabel.stringValue = @"";
        lastHoverLayoutText = @"";
        hideStatusHoverPopover();
        return;
    }
    if ([trimmed isEqualToString:lastHoverLayoutText]) {
        return;
    }

    statusHoverLabel.stringValue = trimmed;
    CGFloat popoverWidth = 460.0;
    CGFloat horizontalPadding = 12.0;
    CGFloat textWidth = popoverWidth - (horizontalPadding * 2.0);
    NSFont *font = statusHoverLabel.font ?: [NSFont systemFontOfSize:[NSFont systemFontSize]];
    NSRect textRect = [trimmed boundingRectWithSize:NSMakeSize(textWidth, CGFLOAT_MAX)
                                            options:(NSStringDrawingUsesLineFragmentOrigin | NSStringDrawingUsesFontLeading)
                                         attributes:@{NSFontAttributeName: font}];
    CGFloat textHeight = MAX(17.0, ceil(textRect.size.height));
    CGFloat popoverHeight = textHeight + 16.0;
    CGFloat labelY = (popoverHeight - textHeight) / 2.0;

    statusHoverLabel.frame = NSMakeRect(horizontalPadding, labelY, textWidth, textHeight);
    statusHoverView.frame = NSMakeRect(0, 0, popoverWidth, popoverHeight);
    statusHoverPopover.contentSize = statusHoverView.frame.size;
    lastHoverLayoutText = trimmed;

    if (statusHoverPopover.isShown && statusItem && statusItem.button) {
        NSRect buttonBounds = statusItem.button.bounds;
        NSRect centeredAnchor = NSMakeRect(NSMidX(buttonBounds), NSMinY(buttonBounds), 1.0, NSHeight(buttonBounds));
        [statusHoverPopover close];
        [statusHoverPopover showRelativeToRect:centeredAnchor
                                        ofView:statusItem.button
                                 preferredEdge:NSRectEdgeMinY];
    }
}

void showStatusHoverPopover(void) {
    return;
}

void updateStatusDetailItem(NSString *detailText) {
    if (!statusDetailItem || !statusDetailView || !statusDetailLabel) {
        return;
    }

    NSString *trimmed = trimmedString(detailText ?: @"");
    if ([trimmed length] == 0 || [trimmed isEqualToString:@"JiraFy Clockwork"]) {
        statusDetailLabel.stringValue = @"";
        statusDetailItem.hidden = YES;
        hideStatusHoverPopover();
        return;
    }

    statusDetailLabel.stringValue = trimmed;
    CGFloat textWidth = 352.0;
    NSFont *font = statusDetailLabel.font ?: [NSFont systemFontOfSize:[NSFont smallSystemFontSize]];
    NSRect textRect = [trimmed boundingRectWithSize:NSMakeSize(textWidth, CGFLOAT_MAX)
                                            options:(NSStringDrawingUsesLineFragmentOrigin | NSStringDrawingUsesFontLeading)
                                         attributes:@{NSFontAttributeName: font}];
    CGFloat textHeight = MAX(17.0, MIN(120.0, ceil(textRect.size.height)));

    statusDetailLabel.frame = NSMakeRect(8, 4, textWidth, textHeight);
    statusDetailView.frame = NSMakeRect(0, 0, 368, textHeight + 8);
    statusDetailItem.hidden = NO;
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

        trayDetailText = @"";
        statusItem.button.toolTip = nil;
        statusHoverView = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 460, 44)];
        statusHoverLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(12, 10, 436, 24)];
        [statusHoverLabel setEditable:NO];
        [statusHoverLabel setBezeled:NO];
        [statusHoverLabel setDrawsBackground:NO];
        [statusHoverLabel setSelectable:NO];
        [statusHoverLabel setLineBreakMode:NSLineBreakByWordWrapping];
        [statusHoverLabel setUsesSingleLineMode:NO];
        [statusHoverLabel setMaximumNumberOfLines:0];
        [statusHoverLabel setAlignment:NSTextAlignmentCenter];
        [(NSTextFieldCell *)statusHoverLabel.cell setWraps:YES];
        [(NSTextFieldCell *)statusHoverLabel.cell setScrollable:NO];
        [(NSTextFieldCell *)statusHoverLabel.cell setTruncatesLastVisibleLine:NO];
        [statusHoverLabel setFont:[NSFont systemFontOfSize:[NSFont systemFontSize]]];
        [statusHoverView addSubview:statusHoverLabel];
        statusHoverPopover = [[NSPopover alloc] init];
        NSViewController *hoverController = [[NSViewController alloc] init];
        hoverController.view = statusHoverView;
        statusHoverPopover.contentViewController = hoverController;
        statusHoverPopover.behavior = NSPopoverBehaviorApplicationDefined;
        statusHoverPopover.animates = NO;

        if (statusItem.button) {
            NSTrackingArea *trackingArea = [[NSTrackingArea alloc] initWithRect:statusItem.button.bounds
                                                                         options:(NSTrackingMouseEnteredAndExited | NSTrackingActiveAlways | NSTrackingInVisibleRect)
                                                                           owner:trayDelegate
                                                                        userInfo:nil];
            [statusItem.button addTrackingArea:trackingArea];
        }

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

        cancelTimerItem = [[NSMenuItem alloc] initWithTitle:@"Cancel Timer"
                                                     action:@selector(cancelTimer:)
                                              keyEquivalent:@""];
        [cancelTimerItem setTarget:trayDelegate];
        cancelTimerItem.hidden = YES;
        [menu addItem:cancelTimerItem];
        updateTimerActionTitle();

        statusDetailItem = [[NSMenuItem alloc] initWithTitle:@"" action:nil keyEquivalent:@""];
        statusDetailView = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 368, 28)];
        statusDetailLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(8, 4, 352, 20)];
        [statusDetailLabel setEditable:NO];
        [statusDetailLabel setBezeled:NO];
        [statusDetailLabel setDrawsBackground:NO];
        [statusDetailLabel setSelectable:NO];
        [statusDetailLabel setLineBreakMode:NSLineBreakByWordWrapping];
        [statusDetailLabel setUsesSingleLineMode:NO];
        [statusDetailLabel setMaximumNumberOfLines:0];
        [statusDetailLabel setFont:[NSFont systemFontOfSize:[NSFont smallSystemFontSize]]];
        [statusDetailLabel setTextColor:[NSColor secondaryLabelColor]];
        [statusDetailView addSubview:statusDetailLabel];
        statusDetailItem.view = statusDetailView;
        statusDetailItem.hidden = YES;
        [menu addItem:statusDetailItem];

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
        updateStatusDetailItem(trayDetailText);

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

void setTrayTooltip(const char *text) {
    NSString *textCopy = (text && text[0] != '\0') ? [NSString stringWithUTF8String:text] : @"";
    dispatch_async(dispatch_get_main_queue(), ^{
        NSString *normalized = trimmedString(textCopy);
        trayDetailText = normalized;
        if (statusItem && statusItem.button) {
            statusItem.button.toolTip = nil;
        }
        updateStatusDetailItem(trayDetailText);
        hideStatusHoverPopover();
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
