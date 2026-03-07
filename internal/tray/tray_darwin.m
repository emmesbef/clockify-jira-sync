#import <Cocoa/Cocoa.h>
#include "tray_darwin.h"

// Go callback declarations (defined in tray_darwin.go via //export)
extern void goTrayShow(void);
extern void goTrayQuit(void);

static NSStatusItem *statusItem = nil;
static NSMenuItem *showHideItem = nil;
static BOOL windowVisible = YES;

@interface TrayDelegate : NSObject
- (void)showWindow:(id)sender;
- (void)showAbout:(id)sender;
- (void)quitApp:(id)sender;
@end

static NSString *appVersion = nil;

@implementation TrayDelegate

- (void)showWindow:(id)sender {
    goTrayShow();
}

- (void)showAbout:(id)sender {
    NSAlert *alert = [[NSAlert alloc] init];
    [alert setMessageText:@"Clockify \u2194 Jira Sync"];
    [alert setInformativeText:[NSString stringWithFormat:@"Version %@\n\nDesktop app to sync Clockify time entries with Jira worklogs.",
                               appVersion ?: @"dev"]];
    [alert setAlertStyle:NSAlertStyleInformational];
    [alert addButtonWithTitle:@"OK"];
    [alert runModal];
}

- (void)quitApp:(id)sender {
    goTrayQuit();
}

@end

static TrayDelegate *trayDelegate = nil;

void updateShowHideTitle(void) {
    if (showHideItem) {
        [showHideItem setTitle:windowVisible ? @"Hide Window" : @"Show Window"];
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

        statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];

        if (iconDataCopy) {
            NSImage *icon = [[NSImage alloc] initWithData:iconDataCopy];
            [icon setSize:NSMakeSize(18, 18)];
            icon.template = YES;
            statusItem.button.image = icon;
        } else {
            statusItem.button.title = @"\u23F1";
        }

        statusItem.button.toolTip = @"Clockify \u2194 Jira Sync";

        NSMenu *menu = [[NSMenu alloc] init];

        showHideItem = [[NSMenuItem alloc] initWithTitle:@"Hide Window"
                                                  action:@selector(showWindow:)
                                           keyEquivalent:@""];
        [showHideItem setTarget:trayDelegate];
        [menu addItem:showHideItem];

        [menu addItem:[NSMenuItem separatorItem]];

        NSString *aboutTitle = [NSString stringWithFormat:@"About (v%@)", appVersion];
        NSMenuItem *aboutItem = [[NSMenuItem alloc] initWithTitle:aboutTitle
                                                           action:@selector(showAbout:)
                                                    keyEquivalent:@""];
        [aboutItem setTarget:trayDelegate];
        [menu addItem:aboutItem];

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
