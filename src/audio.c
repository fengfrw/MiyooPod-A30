#include <SDL2/SDL.h>
#include <SDL2/SDL_mixer.h>
#include <stdlib.h>
#include <string.h>

static Mix_Music *current_music = NULL;
static volatile int music_finished_flag = 0;
static int current_volume = MIX_MAX_VOLUME;
static double cached_duration = 0.0;

// Memory buffer for current track - eliminates SD card I/O during playback
static void *current_music_data = NULL;
static int current_music_size = 0;

static void on_music_finished() {
    music_finished_flag = 1;
}

int audio_init() {
    c_log("audio_init entered");

    c_log("calling Mix_OpenAudio...");
    // 16384 samples (~1.5s latency) - small enough for fast seek, large enough to prevent underruns on ARM
    if (Mix_OpenAudio(44100, MIX_DEFAULT_FORMAT, 2, 16384) < 0) {
        c_logf("Mix_OpenAudio failed: %s", SDL_GetError());
        return -1;
    }
    c_log("Mix_OpenAudio OK");
    // Set hardware DAC volume - Mix_OpenAudio resets it to 0 on A30
    system("amixer sset \"digital volume\" 63 > /dev/null 2>&1");

    int flags = MIX_INIT_MP3 | MIX_INIT_FLAC | MIX_INIT_OGG;
    int initted = Mix_Init(flags);
    if ((initted & MIX_INIT_MP3) == 0)  c_logf("Mix_Init MP3 failed: %s", Mix_GetError());
    else c_log("Mix_Init MP3 OK");
    if ((initted & MIX_INIT_FLAC) == 0) c_logf("Mix_Init FLAC failed: %s", Mix_GetError());
    else c_log("Mix_Init FLAC OK");
    if ((initted & MIX_INIT_OGG) == 0)  c_logf("Mix_Init OGG failed: %s", Mix_GetError());
    else c_log("Mix_Init OGG OK");

    Mix_HookMusicFinished(on_music_finished);
    return 0;
}

// Load from file path (streaming from SD card - fallback)
int audio_load(const char *path) {
    if (current_music) {
        Mix_FreeMusic(current_music);
        current_music = NULL;
    }
    if (current_music_data) {
        free(current_music_data);
        current_music_data = NULL;
    }

    current_music = Mix_LoadMUS(path);
    if (!current_music) {
        c_logf("Mix_LoadMUS failed: %s", Mix_GetError());
        return -1;
    }

    cached_duration = Mix_MusicDuration(current_music);
    { char _dbuf[64]; snprintf(_dbuf, sizeof(_dbuf), "%.2f", cached_duration); c_logf("audio_load OK, duration=", _dbuf); }
    return 0;
}

// Load audio from a memory buffer (data is C-allocated, caller gives ownership).
// Eliminates SD card I/O during playback - SDL_mixer reads from RAM.
// Returns 0 on success, -1 on failure. On failure, caller must NOT free data (we do).
int audio_load_mem(void *data, int size) {
    if (current_music) {
        Mix_FreeMusic(current_music);
        current_music = NULL;
    }
    if (current_music_data) {
        free(current_music_data);
        current_music_data = NULL;
    }

    current_music_data = data;
    current_music_size = size;

    SDL_RWops *rw = SDL_RWFromMem(current_music_data, size);
    if (!rw) {
        c_logf("SDL_RWFromMem failed: %s", SDL_GetError());
        free(current_music_data);
        current_music_data = NULL;
        return -1;
    }

    current_music = Mix_LoadMUS_RW(rw, 1);
    if (!current_music) {
        c_logf("Mix_LoadMUS_RW failed: %s", Mix_GetError());
        free(current_music_data);
        current_music_data = NULL;
        return -1;
    }

    cached_duration = Mix_MusicDuration(current_music);
    { char _dbuf[64]; snprintf(_dbuf, sizeof(_dbuf), "%.2f", cached_duration); c_logf("audio_load_mem OK, duration=", _dbuf); }
    return 0;
}

int audio_play() {
    if (!current_music) return -1;
    music_finished_flag = 0;
    int ret = Mix_PlayMusic(current_music, 0);
    Mix_VolumeMusic(current_volume);
    return ret;
}

void audio_pause() {
    Mix_PauseMusic();
}

void audio_resume() {
    Mix_ResumeMusic();
}

void audio_toggle_pause() {
    if (Mix_PausedMusic()) {
        audio_resume();
    } else {
        audio_pause();
    }
}

void audio_stop() {
    Mix_HaltMusic();
    music_finished_flag = 0;
    cached_duration = 0.0;
}

int audio_is_playing() {
    return Mix_PlayingMusic() && !Mix_PausedMusic();
}

int audio_is_paused() {
    return Mix_PausedMusic();
}

double audio_get_position() {
    if (!current_music || (!Mix_PlayingMusic() && !Mix_PausedMusic())) return 0.0;
    return Mix_GetMusicPosition(current_music);
}

double audio_get_duration() {
    return cached_duration;
}

// Get duration of an audio file without loading it into the player
double audio_get_file_duration(const char *path) {
    Mix_Music *temp_music = Mix_LoadMUS(path);
    if (!temp_music) {
        fprintf(stderr, "Failed to load music for duration: %s - Error: %s\n", path, Mix_GetError());
        return 0.0;
    }
    
    double duration = Mix_MusicDuration(temp_music);
    if (duration < 0) {
        fprintf(stderr, "Mix_MusicDuration failed for: %s - Error: %s\n", path, Mix_GetError());
        duration = 0.0;
    }
    
    Mix_FreeMusic(temp_music);
    
    return duration;
}

int audio_seek(double position) {
    if (!current_music) return -1;
    if (!Mix_PlayingMusic() && !Mix_PausedMusic()) return -1;
    if (position < 0) position = 0;
    if (cached_duration > 0 && position > cached_duration) position = cached_duration;

    int was_paused = Mix_PausedMusic();

    // Flush the ALSA hardware ring buffer by closing and reopening the audio device.
    // Without this, up to ~2s of pre-seek audio drains from ALSA before the seeked
    // audio is heard. Un-hook first to suppress the spurious music_finished event.
    Mix_HookMusicFinished(NULL);
    Mix_HaltMusic();
    Mix_CloseAudio();
    if (Mix_OpenAudio(44100, MIX_DEFAULT_FORMAT, 2, 16384) < 0) {
        Mix_HookMusicFinished(on_music_finished);
        return -1;
    }
    Mix_HookMusicFinished(on_music_finished);
    music_finished_flag = 0;

    // Reload music from the in-memory buffer so mpg123 gets a fresh handle
    // initialized at the correct sample rate (reusing the old handle after
    // CloseAudio causes mpg123 to use a wrong rate, producing 5x position error).
    if (current_music_data && current_music_size > 0) {
        Mix_FreeMusic(current_music);
        current_music = NULL;
        SDL_RWops *rw = SDL_RWFromMem(current_music_data, current_music_size);
        if (rw) {
            current_music = Mix_LoadMUS_RW(rw, 1);
        }
        if (!current_music) {
            return -1;
        }
    }

    // PlayMusic first: mpg123 requires at least one decode callback cycle before
    // Mix_SetMusicPosition will work correctly. Mute to suppress pos-0 audio.
    // SDL_Delay(200) yields the CPU so the audio thread runs at least once before
    // we pause - on A30's slow ARM scheduler PauseMusic can otherwise be called
    // before the callback ever runs, leaving mpg123 cold and causing seeks from
    // near-0 positions to land at the wrong frame.
    Mix_VolumeMusic(0);
    if (Mix_PlayMusic(current_music, 0) < 0) return -1;
    SDL_Delay(200);
    Mix_PauseMusic();
    SDL_LockAudio();
    SDL_UnlockAudio();

    Mix_SetMusicPosition(position);
    Mix_VolumeMusic(current_volume);

    if (!was_paused) {
        Mix_ResumeMusic();
    }
    return 0;
}

void audio_set_volume(int volume) {
    current_volume = volume;
    Mix_VolumeMusic(volume);
}

int audio_check_finished() {
    if (music_finished_flag) {
        music_finished_flag = 0;
        return 1;
    }
    return 0;
}

typedef struct {
    double position;
    double duration;
    int is_playing;
    int is_paused;
    int finished;
} AudioState;

void audio_flush_buffers() {
    // Clear accumulated audio fragments to prevent choppy playback
    // Safe to call during playback - SDL2_mixer will refill from stream
    if (Mix_PlayingMusic() && !Mix_PausedMusic()) {
        SDL_Delay(0); // Yield to allow audio thread to process
    }
}

void audio_get_state(AudioState *state) {
    state->position = 0.0;
    state->duration = cached_duration;
    state->is_playing = 0;
    state->is_paused = 0;
    state->finished = 0;

    if (current_music && Mix_PlayingMusic()) {
        state->position = Mix_GetMusicPosition(current_music);
        state->is_playing = !Mix_PausedMusic();
        state->is_paused = Mix_PausedMusic();
    }

    if (music_finished_flag) {
        music_finished_flag = 0;
        state->finished = 1;
    }
}

// Reinitialize audio device after suspend/resume.
// Closes and reopens SDL_mixer, then reloads the current track from memory if available.
int audio_reinit() {
    double saved_pos = 0.0;
    if (current_music && Mix_PlayingMusic()) {
        saved_pos = Mix_GetMusicPosition(current_music);
    }
    Mix_HaltMusic();
    Mix_CloseAudio();
    if (Mix_OpenAudio(44100, MIX_DEFAULT_FORMAT, 2, 16384) < 0) {
        c_logf("audio_reinit: Mix_OpenAudio failed: %s", SDL_GetError());
        return -1;
    }
    Mix_Init(MIX_INIT_MP3 | MIX_INIT_FLAC | MIX_INIT_OGG);
    Mix_HookMusicFinished(on_music_finished);
    music_finished_flag = 0;
    // Reload and seek if we have music
    if (current_music) {
        if (Mix_PlayMusic(current_music, 0) == 0) {
            Mix_VolumeMusic(current_volume);
            if (saved_pos > 0.5) {
                Mix_SetMusicPosition(saved_pos);
            }
        }
    }
    c_log("audio_reinit OK");
    return 0;
}

void audio_quit() {
    if (current_music) {
        Mix_FreeMusic(current_music);
        current_music = NULL;
    }
    if (current_music_data) {
        free(current_music_data);
        current_music_data = NULL;
    }
    Mix_CloseAudio();
    Mix_Quit();
}
