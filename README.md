# Diablo Helper

Windows only key-input helper for Diablo.

## Features

- Assign every key by clicking a key button, then pressing the target key.
- Press `Esc` during assignment to clear the selected key.
- Start and stop hotkeys control repeated skill input.
- Up to 8 skill keys are supported.
- Each skill has its own interval in milliseconds.
- A bulk interval input can apply the same interval to every skill.
- A hold-style pause key stops skill input only while the key is held.
- Game menu keys stop skill input like the stop key.
- Skill rows can be enabled or disabled individually.
- Settings are saved to and loaded from `settings.toml`.
- Keyboard keys and standard mouse buttons are assignable.
- Start and stop keys cannot be assigned to `Mouse Left`.

## Mouse Button Support

Direct mouse-button support covers:

- `Mouse Left`
- `Mouse Right`
- `Mouse Middle`
- `Mouse X1`
- `Mouse X2`

Extra gaming mouse side buttons beyond `Mouse X1` and `Mouse X2` are not exposed consistently by Windows as normal mouse buttons. They can still be used when the mouse driver maps them to keyboard keys.

`Mouse Left` is supported for assignable actions in general, but not for start and stop keys.

## D3Helper Compatibility Notes

Research baseline:

- [D3Helper Manual](https://d3helper.com/manual)
- [D3Helper usage article](https://canfactory.tistory.com/214)
- [DHelper mouse-button note](https://www.dhelper.co.kr/2021/01/iii.html)

| Area                            | D3Helper behavior                                                                                                                       | This app                                                                                                                    | Status                         |
| ------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- | ------------------------------ |
| Platform                        | Windows desktop helper.                                                                                                                 | Windows-only runtime, with a non-Windows message when built elsewhere.                                                      | Same direction                 |
| Key assignment                  | Click an input field, then press a keyboard or mouse key.                                                                               | Click a key button, then press a keyboard or supported mouse button.                                                        | Same direction                 |
| Assignment cancel               | `Esc` clears the selected D3Helper input.                                                                                               | `Esc` clears the selected key assignment.                                                                                   | Same                           |
| Start key                       | Starts repeated skill input.                                                                                                            | Starts repeated skill input.                                                                                                | Same                           |
| Stop key                        | Stops repeated skill input.                                                                                                             | Stops repeated skill input.                                                                                                 | Same                           |
| Start/stop mouse buttons        | D3Helper documents that left mouse is not available for start/stop.                                                                     | Start and stop keys reject `Mouse Left`; other supported mouse buttons can be assigned.                                     | Same                           |
| Skill keys                      | Up to 8 skill keys with millisecond intervals.                                                                                          | Up to 8 skill keys with per-skill millisecond intervals.                                                                    | Same                           |
| Bulk interval apply             | Provides an interval apply action for skill rows.                                                                                       | Provides a bulk interval field and apply button for every skill row.                                                        | Same direction                 |
| Empty or unused skill rows      | D3Helper rows can be left empty or cleared.                                                                                             | Skill rows can be disabled with a toggle, or left unassigned.                                                               | Different, more explicit       |
| Hold special key                | Holding the special key pauses repeated skill input; releasing it resumes the previous running state.                                   | Holding the pause key pauses repeated skill input only while running; releasing it resumes if the stop key was not pressed. | Same                           |
| Number of hold special keys     | D3Helper UI shows one special key.                                                                                                      | One pause key is supported.                                                                                                 | Same                           |
| Game menu stop keys             | D3Helper stops repeated input for game menu actions such as inventory, skill, follower, map, world map, town portal, chat, and whisper. | Inventory, skill, follower, map, world map, town portal, chat, and whisper stop repeated input.                             | Same                           |
| World map and whisper stop keys | D3Helper-style UI includes separate world map and whisper keys.                                                                         | Separate world map and whisper stop-key fields are supported.                                                               | Same                           |
| Save and load                   | Saved values can be reused. The referenced usage article describes changing saved values by class.                                      | Saves and loads one `settings.toml` file next to the executable.                                                            | Partial                        |
| Multiple profiles               | D3Helper UI commonly shows a profile name such as `default`.                                                                            | No profile selector or multiple TOML profile manager yet.                                                                   | Not supported                  |
| Mouse middle button             | D3Helper documents mouse input, including wheel-related assignment.                                                                     | `Mouse Middle` is directly supported.                                                                                       | Same for middle click          |
| Mouse wheel scroll              | D3Helper documentation says mouse wheel input is possible.                                                                              | Wheel-up and wheel-down scroll events are not assignable yet.                                                               | Not supported                  |
| Side mouse buttons              | Related helper documentation calls out 5-button mouse support through `XBUTTON1` and `XBUTTON2`.                                        | `Mouse X1` and `Mouse X2` are directly supported. Extra side buttons require driver-level keyboard mapping.                 | Same for standard side buttons |
| Runtime technique               | Related helper documentation describes Windows keyboard/mouse event and hook usage, not memory or packet manipulation.                  | Uses Windows keyboard/mouse hooks and input events only.                                                                    | Same direction                 |

## Build

```powershell
go build -o dist\diablo-helper.exe .\cmd\diablo-helper
```

Cross-build from macOS or Linux:

```sh
GOOS=windows GOARCH=amd64 go build -o dist/diablo-helper.exe ./cmd/diablo-helper
```

## Usage

1. Run `diablo-helper.exe` on Windows.
2. Click a key button and press the key to assign.
3. Set skill intervals in milliseconds.
4. Use `Save TOML` to write `settings.toml` next to the executable.
5. Use the assigned start key to begin skill input.
6. Use the stop key or one of the game menu keys to stop skill input.
7. Hold the pause key to suspend skill input temporarily.
