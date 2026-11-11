Unicode true
!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "x64.nsh"

!define APP "ProtosVPN"
!define VERSION "0.1.2"
!define UNINSTALL_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\ProtosVPN"
!define WEBVIEW_KEY "Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}"
Name "${APP} ${VERSION}-preview"
OutFile "../../release/ProtosVPN-${VERSION}-preview-windows-amd64-setup.exe"
InstallDir "$LOCALAPPDATA\Programs\ProtosVPN"
InstallDirRegKey HKCU "Software\ProtosVPN" "InstallDir"
RequestExecutionLevel user
SetCompressor /SOLID lzma
ShowInstDetails show
ShowUninstDetails show
VIProductVersion "${VERSION}.0"
VIAddVersionKey /LANG=1033 "ProductName" "ProtosVPN Setup"
VIAddVersionKey /LANG=1033 "FileDescription" "ProtosVPN per-user installer"
VIAddVersionKey /LANG=1033 "FileVersion" "${VERSION}"
VIAddVersionKey /LANG=1033 "LegalCopyright" "ProtosVPN"
!define MUI_ICON "icon.ico"
!define MUI_UNICON "icon.ico"
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TEXT "$(WelcomeText)"
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_FUNCTION LaunchApp
!define MUI_FINISHPAGE_RUN_TEXT "$(LaunchText)"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH
!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "Russian"

LangString WelcomeText ${LANG_ENGLISH} "This wizard installs ProtosVPN for the current Windows user. Administrator rights are not required.$\r$\n$\r$\nDevelopment preview: VPN connections are not available yet. Profiles are kept only in memory and disappear when the application exits.$\r$\n$\r$\nClose ProtosVPN before installing or updating."
LangString WelcomeText ${LANG_RUSSIAN} "Мастер установит ProtosVPN для текущего пользователя Windows. Права администратора не требуются.$\r$\n$\r$\nПредварительная версия: VPN-подключение пока недоступно. Профили хранятся только в памяти и исчезают после закрытия.$\r$\n$\r$\nПеред установкой или обновлением закройте ProtosVPN."
LangString AppSection ${LANG_ENGLISH} "Application and Start menu shortcut (required)"
LangString AppSection ${LANG_RUSSIAN} "Приложение и ярлык в меню «Пуск» (обязательно)"
LangString DesktopSection ${LANG_ENGLISH} "Desktop shortcut"
LangString DesktopSection ${LANG_RUSSIAN} "Ярлык на рабочем столе"
LangString LaunchText ${LANG_ENGLISH} "Launch ProtosVPN"
LangString LaunchText ${LANG_RUSSIAN} "Запустить ProtosVPN"
LangString NeedX64 ${LANG_ENGLISH} "This installer requires 64-bit Windows."
LangString NeedX64 ${LANG_RUSSIAN} "Для этой сборки требуется 64-разрядная Windows."
LangString MissingWebView ${LANG_ENGLISH} "Microsoft Edge WebView2 Runtime was not found. ProtosVPN requires it to open. Open the official Microsoft download page? Install the Evergreen Runtime, then launch ProtosVPN from the Start menu."
LangString MissingWebView ${LANG_RUSSIAN} "Microsoft Edge WebView2 Runtime не найден. Без него приложение не откроется. Открыть официальную страницу Microsoft? Установите Evergreen Runtime, затем запустите ProtosVPN из меню «Пуск»."
LangString RemoveFailed ${LANG_ENGLISH} "Could not remove ProtosVPN.exe. Close the application, then retry uninstalling."
LangString RemoveFailed ${LANG_RUSSIAN} "Не удалось удалить ProtosVPN.exe. Закройте приложение и повторите удаление."
Var WebViewVersion

Function .onInit
 SetShellVarContext current
 !insertmacro MUI_LANGDLL_DISPLAY
 ${IfNot} ${RunningX64}
  MessageBox MB_ICONSTOP "$(NeedX64)"
  Abort
 ${EndIf}
FunctionEnd

; Evergreen WebView2 can be registered per-machine (32/64-bit views) or per-user.
Function CheckWebView
 StrCpy $WebViewVersion ""
 SetRegView 32
 ReadRegStr $WebViewVersion HKLM "${WEBVIEW_KEY}" "pv"
 ${If} $WebViewVersion == ""
 ${OrIf} $WebViewVersion == "0.0.0.0"
  SetRegView 64
  ReadRegStr $WebViewVersion HKLM "${WEBVIEW_KEY}" "pv"
 ${EndIf}
 ${If} $WebViewVersion == ""
 ${OrIf} $WebViewVersion == "0.0.0.0"
  SetRegView 32
  ReadRegStr $WebViewVersion HKCU "${WEBVIEW_KEY}" "pv"
 ${EndIf}
 ${If} $WebViewVersion == "0.0.0.0"
  StrCpy $WebViewVersion ""
 ${EndIf}
 SetRegView 32
FunctionEnd

Section "$(AppSection)" SEC_APP
 SectionIn RO
 SetShellVarContext current
 SetOutPath "$INSTDIR"
 File /oname=ProtosVPN.exe "../bin/ProtosVPN-windows-amd64.exe"
 File /oname=README.ru.md "../../README.md"
 File /oname=VERIFICATION.md "../../VERIFICATION.md"
 WriteUninstaller "$INSTDIR\Uninstall.exe"
 CreateDirectory "$SMPROGRAMS\ProtosVPN"
 CreateShortcut "$SMPROGRAMS\ProtosVPN\ProtosVPN.lnk" "$INSTDIR\ProtosVPN.exe"
 CreateShortcut "$SMPROGRAMS\ProtosVPN\Uninstall.lnk" "$INSTDIR\Uninstall.exe"
 WriteRegStr HKCU "Software\ProtosVPN" "InstallDir" "$INSTDIR"
 WriteRegStr HKCU "${UNINSTALL_KEY}" "DisplayName" "ProtosVPN (development preview)"
 WriteRegStr HKCU "${UNINSTALL_KEY}" "DisplayVersion" "${VERSION}-preview"
 WriteRegStr HKCU "${UNINSTALL_KEY}" "Publisher" "ProtosVPN"
 WriteRegStr HKCU "${UNINSTALL_KEY}" "InstallLocation" "$INSTDIR"
 WriteRegStr HKCU "${UNINSTALL_KEY}" "DisplayIcon" "$INSTDIR\ProtosVPN.exe"
 WriteRegStr HKCU "${UNINSTALL_KEY}" "UninstallString" '$\"$INSTDIR\Uninstall.exe$\"'
 WriteRegDWORD HKCU "${UNINSTALL_KEY}" "NoModify" 1
 WriteRegDWORD HKCU "${UNINSTALL_KEY}" "NoRepair" 1
SectionEnd

Section /o "$(DesktopSection)" SEC_DESKTOP
 SetShellVarContext current
 CreateShortcut "$DESKTOP\ProtosVPN.lnk" "$INSTDIR\ProtosVPN.exe"
SectionEnd

Function LaunchApp
 Call CheckWebView
 ${If} $WebViewVersion == ""
  MessageBox MB_ICONINFORMATION|MB_YESNO "$(MissingWebView)" IDNO done
  ExecShell "open" "https://developer.microsoft.com/microsoft-edge/webview2/"
 ${Else}
  Exec '"$INSTDIR\ProtosVPN.exe"'
 ${EndIf}
 done:
FunctionEnd

Function .onInstSuccess
 ; Check even when the user does not choose to launch at the end.
 Call CheckWebView
 ${If} $WebViewVersion == ""
  DetailPrint "Microsoft Edge WebView2 Runtime is required; install it from Microsoft."
 ${EndIf}
FunctionEnd

Section "Uninstall"
 SetShellVarContext current
 ClearErrors
 Delete "$INSTDIR\ProtosVPN.exe"
 ${If} ${Errors}
  MessageBox MB_ICONSTOP "$(RemoveFailed)"
  Abort
 ${EndIf}
 Delete "$DESKTOP\ProtosVPN.lnk"
 Delete "$SMPROGRAMS\ProtosVPN\ProtosVPN.lnk"
 Delete "$SMPROGRAMS\ProtosVPN\Uninstall.lnk"
 RMDir "$SMPROGRAMS\ProtosVPN"
 Delete "$INSTDIR\README.ru.md"
 Delete "$INSTDIR\VERIFICATION.md"
 Delete "$INSTDIR\Uninstall.exe"
 ; Do not recursively remove a user-chosen folder or WebView preferences.
 RMDir "$INSTDIR"
 DeleteRegKey HKCU "${UNINSTALL_KEY}"
 DeleteRegValue HKCU "Software\ProtosVPN" "InstallDir"
 DeleteRegKey /ifempty HKCU "Software\ProtosVPN"
SectionEnd
