#pragma once
int  updater_init(void);
int  updater_refresh(unsigned char *pixels);
int  updater_poll_event(void);
void updater_quit(void);
extern const int UPDATER_RENDER_WIDTH;
extern const int UPDATER_RENDER_HEIGHT;
