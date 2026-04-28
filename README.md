# Diablo Helper

`Diablo Helper`는 Diablo IV 플레이 중 반복 키 입력을 보조하는 Windows 전용 GUI 앱입니다. 전역 키보드/마우스 훅으로 시작, 종료, 일시정지, 메뉴 키를 감지하고, 설정한 기술 키와 클릭 반복 키를 `SendInput`으로 주기적으로 전송합니다.

## 지원 환경

- Windows 전용입니다.
- Go 모듈 버전은 `1.26.2`입니다.
- 런타임 의존성은 Go 표준 라이브러리와 Win32 API뿐입니다.
- 키보드 키와 Windows가 표준 버튼으로 노출하는 마우스 입력을 지원합니다.

## 주요 기능

- 전역 시작 키와 종료 키를 직접 지정할 수 있습니다.
- 기술 반복은 최대 8개 슬롯을 지원합니다.
- 각 기술 슬롯마다 사용 여부, 출력 키, 실행 간격을 개별 설정할 수 있습니다.
- `일괄 간격`과 `키별 간격`으로 1번부터 8번 기술 간격을 한 번에 채울 수 있습니다.
- 일시정지 키는 누르고 있는 동안에만 기술 반복을 멈춥니다.
- 클릭 반복은 기술 반복과 별도 러너로 동작하며 자체 시작 키, 종료 키, 출력 키, 간격을 가집니다.
- 게임 메뉴 키 10개를 따로 등록할 수 있고, 이 키를 누르면 기술 반복과 클릭 반복이 함께 정지합니다.
- 설정을 `.toml` 파일로 저장하고 다시 불러올 수 있습니다.
- 실행 파일과 같은 폴더에 `default.toml`이 있으면 시작 시 자동 로드합니다.
- 실행 가능한 기술이 하나도 없으면 기술 반복은 시작되지 않습니다.
- 클릭 반복 출력 키가 없거나 간격이 유효하지 않으면 클릭 반복도 시작되지 않습니다.

## 입력 규칙

- 키 캡처 중 아무 키나 누르면 해당 바인딩에 즉시 저장됩니다.
- 키 캡처 중 `Esc`를 누르면 해당 바인딩이 해제됩니다.
- 같은 키를 누른 채 반복 입력되는 `key down` 이벤트는 `key up` 전까지 한 번만 처리합니다.
- 지원 마우스 버튼은 `Mouse Left`, `Mouse Right`, `Mouse Middle`, `Mouse X1`, `Mouse X2`입니다.
- `Mouse Left`는 기술 키, 일시정지 키, 게임 메뉴 키, 클릭 반복 출력 키에는 사용할 수 있습니다.
- `Mouse Left`는 기술 반복 시작/종료 키와 클릭 반복 시작/종료 키에는 사용할 수 없습니다.
- 기술 키와 클릭 반복 출력 키에는 `Shift`, `Ctrl`, `Alt`, `Pause`, `Caps Lock`, `Esc`, `Left Win`, `Right Win`을 사용할 수 없습니다.
- 시작/종료/일시정지 같은 핫키에는 위 시스템 키를 지정할 수 있습니다.

## 기본 설정

- 시작 키: 미지정
- 종료 키: 미지정
- 일시정지 키: `Mouse Right`
- 게임 메뉴 키 `캐릭터`: `C`
- 게임 메뉴 키 `스킬 배치`: `S`
- 게임 메뉴 키 `능력치`: `A`
- 게임 메뉴 키 `지도`: `Tab`
- 게임 메뉴 키 `일지`: `J`
- 게임 메뉴 키 `소셜`: `O`
- 게임 메뉴 키 `클랜`: `N`
- 게임 메뉴 키 `차원문`: `T`
- 게임 메뉴 키 `컬렉션`: `Y`
- 게임 메뉴 키 `상점`: `P`
- 기술 1~8: 기본 키 미지정, 기본 간격 `1000ms`, 기본 사용 여부 `false`
- 키별 간격: `0ms`
- 클릭 반복 출력 키: `Mouse Left`
- 클릭 반복 간격: `100ms`
- 클릭 반복 시작 키: 미지정
- 클릭 반복 종료 키: 미지정

## 주의 사항

- 입력은 Diablo IV 창으로 한정되지 않고 현재 활성 창으로 전송됩니다.
- 기술 반복이나 클릭 반복이 실행 중일 때 다른 창으로 포커스를 옮기면 그 창에도 입력이 들어갈 수 있습니다.
- 다른 작업으로 전환하기 전에 종료 키 또는 게임 메뉴 키로 먼저 정지하는 것이 안전합니다.

## 설정 파일

- 설정 형식은 외부 TOML 라이브러리 없이 직접 구현한 파서와 serializer를 사용합니다.
- 설정 파일 크기 제한은 `64 KiB`입니다.
- 앱이 저장하는 대표 키는 `start_key_name`, `start_key_vk`, `stop_key_name`, `stop_key_vk`, `pause_key_name`, `pause_key_vk`, `skill_gap_ms`, `clicker_*`, `menu_*` 입니다.
- 기술은 `[[skills]]` 배열로 최대 8개까지 저장합니다.
- 문자열은 반드시 큰따옴표로 감싸야 합니다.
- 저장/불러오기 과정에서 `key_name`은 `key_vk` 기준의 canonical name으로 정규화됩니다.
- 범위를 벗어난 일부 값은 기본값 또는 미지정 상태로 보정됩니다.
- 알 수 없는 키/섹션, 잘못된 문자열/정수/불리언 형식, 8개를 넘는 `[[skills]]`는 오류로 거부됩니다.

예시:

```toml
start_key_name = "F5"
start_key_vk = 116
stop_key_name = "F6"
stop_key_vk = 117
pause_key_name = "Mouse Right"
pause_key_vk = 2
skill_gap_ms = 0

clicker_start_key_name = ""
clicker_start_key_vk = 0
clicker_stop_key_name = ""
clicker_stop_key_vk = 0
clicker_key_name = "Mouse Left"
clicker_key_vk = 1
clicker_interval_ms = 100

[[skills]]
name = "Skill 1"
key_name = "1"
key_vk = 49
interval_ms = 1000
enabled = true
```

## 프로젝트 구조

- `cmd/diablo-helper`: 플랫폼별 `main` 진입점입니다.
- `internal/app`: Win32 창, 전역 훅, 키 캡처, UI, 파일 대화상자, 입력 러너, `SendInput` 처리를 담당합니다.
- `internal/config`: 설정 모델, 정규화/검증, TOML marshal/parse, 파일 저장/불러오기를 담당합니다.
- `internal/meta`: 버전, 커밋, 빌드 날짜, Go 버전 같은 빌드 메타데이터를 보관합니다.
- `dist`: Windows 빌드 산출물 경로입니다.

## 메타데이터

- 작성자: `dongju93`
- 저장소: `https://github.com/dongju93/diablo-helper`
- 창 제목은 `Diablo Helper v<Version>` 형식입니다.
- 소스 코드 기본값은 `GoVersion=1.26.2` 만 지정되어 있으며 `Version` 은 매 Release 마다 갱신됩니다, `Commit`, `BuildDate` 는 `ldflags`로 빌드 시 갱신합니다.

## 빌드

사전 준비:

```powershell
go install github.com/tc-hib/go-winres@latest
```

Windows에서 빌드:

```powershell
$commit=$(git rev-parse --short HEAD); $date=$(Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"); $gover=(go version) -replace 'go version go','' -replace ' .*',''; go-winres make --in cmd\diablo-helper\winres\winres.json --out cmd\diablo-helper\rsrc; go build -ldflags "-H=windowsgui -X github.com/dongju93/diablo-helper/internal/meta.Commit=$commit -X github.com/dongju93/diablo-helper/internal/meta.BuildDate=$date -X github.com/dongju93/diablo-helper/internal/meta.GoVersion=$gover" -o dist\diablo-helper.exe .\cmd\diablo-helper
```

macOS 또는 Linux에서 Windows 실행 파일로 크로스 빌드:

```sh
COMMIT=$(git rev-parse --short HEAD); DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ"); GOVER=$(go version | sed -E 's/^go version go([^ ]+).*/\1/'); go-winres make --in cmd/diablo-helper/winres/winres.json --out cmd/diablo-helper/rsrc; GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui -X github.com/dongju93/diablo-helper/internal/meta.Commit=$COMMIT -X github.com/dongju93/diablo-helper/internal/meta.BuildDate=$DATE -X github.com/dongju93/diablo-helper/internal/meta.GoVersion=$GOVER" -o dist/diablo-helper.exe ./cmd/diablo-helper
```

## 테스트

```powershell
go test ./...
go test -v ./internal/config
go test -v -run TestDefaultConfig ./internal/config
```

## 사용 방법

1. Windows에서 `diablo-helper.exe`를 실행합니다.
2. 바인딩 버튼을 눌러 시작, 종료, 일시정지, 기술, 클릭 반복, 게임 메뉴 키를 지정합니다.
3. 사용할 기술 슬롯의 토글을 켜고 실행 간격을 밀리초 단위로 입력합니다.
4. 필요하면 `일괄 간격`과 `키별 간격`을 입력한 뒤 `일괄 적용`으로 1~8번 기술 간격을 채웁니다.
5. 클릭 반복을 쓰려면 시작 키, 종료 키, 출력 키, 간격을 설정합니다.
6. 필요하면 `저장하기`로 `.toml` 설정을 저장하고 `불러오기`로 다시 읽습니다.
7. 기술 반복은 시작 키로, 클릭 반복은 클릭 반복 시작 키로 실행합니다.
8. 종료 키, 클릭 반복 종료 키, 또는 게임 메뉴 키로 동작을 멈춥니다.
9. 일시정지 키는 누르고 있는 동안에만 기술 반복을 멈추며 클릭 반복에는 적용되지 않습니다.
