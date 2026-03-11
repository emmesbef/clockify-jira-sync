# Intro
The App needs an update, it has several bugs and has 2 feature requests. Here in summary what have to be done
* Projects are not correctly loaded
* Auto Update toggle does not work correctly
* Systemtray Icon needs an update
* New Dialog on clicking on the Systemtray Icon
* App Icon does not need to stay in the Menubar

## Implementation Status
- ✅ Projects reload immediately after saving settings (no restart required).
- ✅ Auto-update toggles persist correctly and update checks respect the configured values.
- ✅ macOS tray now shows live timer + ticket key, with configurable `hh:mm` / `hh:mm:ss` format.
- ✅ macOS tray menu includes **Start Timer…** and now opens a standalone tray popover (no main window foregrounding).
- ✅ Tray timer menu action now toggles dynamically: **Start Timer…** when idle and **Stop Timer** while a timer is running.
- ✅ Tray start popover uses a single description input with top-5 assigned tickets on empty/focus and Jira search while typing.
- ✅ Close-to-tray behavior now enforces tray-only mode while closed (hidden from Dock and Cmd+Tab) until **Show Window** is explicitly chosen from tray.
- ✅ macOS `Cmd+Q` and window close (`x`) now follow the same hide-to-tray behavior; explicit tray **Quit** remains the true app exit.
- ✅ Timer UI state now resyncs on window restore and clears stale running state when backend timer is already stopped.

# Details
## Projects are not correctly loaded
As soon the Settings are save on first setup, the Projects are not correctly loaded, the App needs to be restarted
## Auto Update toggle does not work correctly
The Auto Update toggle has no effect it is also saved as false in the env-file. Also As soon as the toggle is on, no updates are done
## Systemtray Icon needs an update
The Systemtray Icon needs to be updated. It should show besides a clock, a timmer which could be configured in the settings in 2 forms hh:mm and hh:mm:ss. Besides the timer ticket-key should be visible in the Systemtray
## New Dialog on clicking on the Systemtray Icon
When clicking on the Systemtray Icon a new Menu Entry should be listed. Start-Timer clicking on this opens a new Modal where the same can be done as in the app. This means, ticket can be searched, project be choosen and so on
## App Icon does not need to stay in the Menubar
It's enough when the Icon in the System Tray is shown, the app does not need to be open all time and the icon does not need to be in the menubar the whole time
