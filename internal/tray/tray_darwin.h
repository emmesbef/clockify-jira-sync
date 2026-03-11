#ifndef TRAY_DARWIN_H
#define TRAY_DARWIN_H

void initTray(const char *version, const void *iconData, int iconLen);
void setTrayWindowVisible(int visible);
void setTrayStatusText(const char *text);
void setTrayTimerRunning(int running);
void setTrayAppBackgroundMode(void);
void setTrayAppForegroundMode(void);

#endif
