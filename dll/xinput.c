/* XInput shim for Wine/CrossOver: serves the gamepad state that raikirid
 * writes to /tmp/raikiri.xinput (Z:\tmp\raikiri.xinput inside the bottle). */
#define _WIN32_WINNT 0x0602
#include <windows.h>
#include <xinput.h>
#include <stdint.h>
#include <string.h>

#define GUIDE_BUTTON 0x0400

/* Must match the layout in main.go. */
#pragma pack(push, 1)
typedef struct {
    uint32_t packet;
    uint8_t connected;
    uint8_t pad[3];
    XINPUT_GAMEPAD gamepad;
    uint16_t rumble_left;
    uint16_t rumble_right;
} shared_t;
#pragma pack(pop)

static volatile shared_t *shm;

/* ponytail: lazy init without a lock, a race only leaks one extra mapping. */
static volatile shared_t *map_state(void)
{
    if (shm)
        return shm;
    HANDLE f = CreateFileW(L"Z:\\tmp\\raikiri.xinput", GENERIC_READ | GENERIC_WRITE,
                           FILE_SHARE_READ | FILE_SHARE_WRITE, NULL, OPEN_EXISTING, 0, NULL);
    if (f == INVALID_HANDLE_VALUE)
        return NULL;
    HANDLE m = CreateFileMappingW(f, NULL, PAGE_READWRITE, 0, sizeof(shared_t), NULL);
    CloseHandle(f);
    if (!m)
        return NULL;
    shm = MapViewOfFile(m, FILE_MAP_ALL_ACCESS, 0, 0, sizeof(shared_t));
    CloseHandle(m);
    return shm;
}

static volatile shared_t *connected(DWORD user)
{
    volatile shared_t *s = user == 0 ? map_state() : NULL;
    return s && s->connected ? s : NULL;
}

/* Ordinal 100: like XInputGetState but reports the Guide button too. */
DWORD WINAPI XInputGetStateEx(DWORD user, XINPUT_STATE *state)
{
    volatile shared_t *s = connected(user);
    if (!s)
        return ERROR_DEVICE_NOT_CONNECTED;
    state->dwPacketNumber = s->packet;
    /* ponytail: no seqlock, a torn frame of gamepad state is harmless. */
    memcpy(&state->Gamepad, (const void *)&s->gamepad, sizeof(state->Gamepad));
    return ERROR_SUCCESS;
}

DWORD WINAPI XInputGetState(DWORD user, XINPUT_STATE *state)
{
    DWORD r = XInputGetStateEx(user, state);
    if (r == ERROR_SUCCESS)
        state->Gamepad.wButtons &= ~GUIDE_BUTTON;
    return r;
}

DWORD WINAPI XInputSetState(DWORD user, XINPUT_VIBRATION *vibration)
{
    volatile shared_t *s = connected(user);
    if (!s)
        return ERROR_DEVICE_NOT_CONNECTED;
    s->rumble_left = vibration->wLeftMotorSpeed;
    s->rumble_right = vibration->wRightMotorSpeed;
    return ERROR_SUCCESS;
}

DWORD WINAPI XInputGetCapabilities(DWORD user, DWORD flags, XINPUT_CAPABILITIES *caps)
{
    (void)flags;
    if (!connected(user))
        return ERROR_DEVICE_NOT_CONNECTED;
    memset(caps, 0, sizeof(*caps));
    caps->Type = XINPUT_DEVTYPE_GAMEPAD;
    caps->SubType = XINPUT_DEVSUBTYPE_GAMEPAD;
    caps->Flags = XINPUT_CAPS_FFB_SUPPORTED | XINPUT_CAPS_WIRELESS;
    caps->Gamepad.wButtons = 0xf3ff;
    caps->Gamepad.bLeftTrigger = caps->Gamepad.bRightTrigger = 0xff;
    caps->Gamepad.sThumbLX = caps->Gamepad.sThumbLY = (SHORT)0xffc0;
    caps->Gamepad.sThumbRX = caps->Gamepad.sThumbRY = (SHORT)0xffc0;
    caps->Vibration.wLeftMotorSpeed = caps->Vibration.wRightMotorSpeed = 0xff;
    return ERROR_SUCCESS;
}

void WINAPI XInputEnable(BOOL enable)
{
    (void)enable;
}

DWORD WINAPI XInputGetBatteryInformation(DWORD user, BYTE dev_type, XINPUT_BATTERY_INFORMATION *info)
{
    (void)dev_type;
    if (!connected(user))
        return ERROR_DEVICE_NOT_CONNECTED;
    info->BatteryType = BATTERY_TYPE_WIRED;
    info->BatteryLevel = BATTERY_LEVEL_FULL;
    return ERROR_SUCCESS;
}

DWORD WINAPI XInputGetKeystroke(DWORD user, DWORD reserved, XINPUT_KEYSTROKE *keystroke)
{
    (void)reserved;
    (void)keystroke;
    return connected(user) ? ERROR_EMPTY : ERROR_DEVICE_NOT_CONNECTED;
}

DWORD WINAPI XInputGetDSoundAudioDeviceGuids(DWORD user, GUID *render, GUID *capture)
{
    if (!connected(user))
        return ERROR_DEVICE_NOT_CONNECTED;
    memset(render, 0, sizeof(*render));
    memset(capture, 0, sizeof(*capture));
    return ERROR_SUCCESS;
}

DWORD WINAPI XInputGetAudioDeviceIds(DWORD user, LPWSTR render, UINT *render_count,
                                     LPWSTR capture, UINT *capture_count)
{
    (void)render;
    (void)capture;
    if (!connected(user))
        return ERROR_DEVICE_NOT_CONNECTED;
    if (render_count)
        *render_count = 0;
    if (capture_count)
        *capture_count = 0;
    return ERROR_SUCCESS;
}
