"""Static regression checks; does not replace an actual Windows installation test."""
from pathlib import Path
import re
import unittest

SCRIPT = (Path(__file__).parent / 'windows' / 'installer.nsi').read_text()

class InstallerTests(unittest.TestCase):
    def test_both_shortcuts_are_required(self):
        section = re.search(r'Section "\$\(AppSection\)" SEC_APP(.*?)SectionEnd', SCRIPT, re.S).group(1)
        self.assertIn('SectionIn RO', section)
        for location in ('$SMPROGRAMS', '$DESKTOP'):
            self.assertIn(f'CreateShortcut "{location}\\ProtosVPN.lnk" "$INSTDIR\\ProtosVPN.exe" "" "$INSTDIR\\ProtosVPN.exe" 0', section)
        self.assertNotIn('Section /o', SCRIPT)
        self.assertIn('${If} ${Errors}', section)
        self.assertIn('$(ShortcutFailed)', section)

    def test_current_user_and_uninstall_cleanup(self):
        self.assertIn('RequestExecutionLevel user', SCRIPT)
        self.assertIn('SetShellVarContext current', SCRIPT)
        uninstall = SCRIPT.split('Section "Uninstall"', 1)[1]
        for location in ('$SMPROGRAMS', '$DESKTOP'):
            self.assertIn(f'Delete "{location}\\ProtosVPN.lnk"', uninstall)
        self.assertNotIn('RMDir /r', SCRIPT)

    def test_installer_version_matches_application(self):
        import json
        root = Path(__file__).parent.parent
        version = json.loads((root / 'wails.json').read_text())['info']['productVersion']
        self.assertIn(f'!define VERSION "{version}"', SCRIPT)

if __name__ == '__main__':
    unittest.main()
