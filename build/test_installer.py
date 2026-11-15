"""Installer checks and isolated download/package tests; not a Windows install test."""
from pathlib import Path
from contextlib import redirect_stdout
import io
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

    def test_both_core_payloads_and_licenses_are_installed_and_removed(self):
        install = SCRIPT.split('Section "$(AppSection)" SEC_APP', 1)[1].split('SectionEnd', 1)[0]
        uninstall = SCRIPT.split('Section "Uninstall"', 1)[1]
        for name in ('xray.exe', 'xray-legacy.exe'):
            self.assertIn(f'File /oname={name} "../cores/windows-amd64/{name}"', install)
            self.assertIn(f'Delete "$INSTDIR\\core\\{name}"', uninstall)
        for name in ('XRAY-LICENSE.txt', 'XRAY-LEGACY-LICENSE.txt', 'manifest.json'):
            self.assertIn(f'File /oname={name} "../cores/{name}"', install)
            self.assertIn(f'Delete "$INSTDIR\\core\\{name}"', uninstall)
        self.assertIn('THIRD-PARTY.md', install)

    def test_installer_version_matches_application(self):
        import json
        root = Path(__file__).parent.parent
        version = json.loads((root / 'wails.json').read_text())['info']['productVersion']
        self.assertIn(f'!define VERSION "{version}"', SCRIPT)

class CoreDownloadTests(unittest.TestCase):
    def test_fetches_verified_dual_pinned_cores_with_modern_records_first(self):
        import hashlib
        import importlib.util
        import io
        import json
        import tempfile
        import zipfile
        from unittest.mock import patch

        spec = importlib.util.spec_from_file_location('fetch_xray', Path(__file__).parent / 'fetch-xray.py')
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        downloads = {}
        for version in ('v26.11.9', 'v25.12.8'):
            for platform, archive, binary in (
                ('linux-amd64', 'Xray-linux-64.zip', 'xray'),
                ('windows-amd64', 'Xray-windows-64.zip', 'xray.exe'),
            ):
                data = io.BytesIO()
                with zipfile.ZipFile(data, 'w') as z:
                    z.writestr(binary, f'{platform} {version}')
                url = f'https://github.com/XTLS/Xray-core/releases/download/{version}/{archive}'
                downloads[url] = data.getvalue()
                downloads[url + '.dgst'] = ('SHA256= ' + hashlib.sha256(data.getvalue()).hexdigest()).encode()
            downloads[f'https://raw.githubusercontent.com/XTLS/Xray-core/{version}/LICENSE'] = version.encode()
        with tempfile.TemporaryDirectory() as directory, patch.object(module, 'ROOT', Path(directory)), patch.object(module, 'fetch', side_effect=downloads.__getitem__), redirect_stdout(io.StringIO()):
            module.main()
            cores = Path(directory) / 'build/cores'
            records = json.loads((cores / 'manifest.json').read_text())
            self.assertEqual(len(records), 4)
            for platform, binary in (('windows-amd64', 'xray.exe'), ('linux-amd64', 'xray')):
                self.assertEqual(next(r for r in records if r['platform'] == platform)['variant'], 'modern')
                for variant, version, filename in (('modern', 'v26.11.9', binary), ('legacy', 'v25.12.8', binary.replace('xray', 'xray-legacy'))):
                    record = next(r for r in records if r['platform'] == platform and r['variant'] == variant)
                    self.assertEqual(record['version'], version)
                    self.assertIn(f'/download/{version}/', record['source'])
                    self.assertEqual(record['binarySHA256'], hashlib.sha256((cores / platform / filename).read_bytes()).hexdigest())
                    self.assertEqual(len(record['archiveSHA256']), 64)
            self.assertEqual((cores / 'XRAY-LICENSE.txt').read_text(), 'v26.11.9')
            self.assertEqual((cores / 'XRAY-LEGACY-LICENSE.txt').read_text(), 'v25.12.8')
            bad = next(url for url in downloads if '/v25.12.8/' in url and url.endswith('.dgst'))
            downloads[bad] = b'SHA256= ' + b'0' * 64
            with self.assertRaisesRegex(RuntimeError, 'SHA256 mismatch'):
                module.main()


class PackagingTests(unittest.TestCase):
    def setUp(self):
        import hashlib
        import importlib.util
        import json
        import tempfile
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        spec = importlib.util.spec_from_file_location('package_xray', Path(__file__).parent / 'package.py')
        self.module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.module)
        self.module.ROOT = self.root
        self.module.OUT = self.root / 'release'
        files = {'build/bin/ProtosVPN-windows-amd64.exe': b'MZfixture',
                 'build/bin/ProtosVPN-linux-amd64': b'\x7fELFfixture',
                 'README.md': b'readme', 'VERIFICATION.md': b'verification', 'THIRD-PARTY.md': b'notices',
                 'build/cores/XRAY-LICENSE.txt': b'modern license',
                 'build/cores/XRAY-LEGACY-LICENSE.txt': b'legacy license',
                 'build/cores/windows-amd64/xray.exe': b'modern executable',
                 'build/cores/windows-amd64/xray-legacy.exe': b'legacy executable'}
        for name, data in files.items():
            path = self.root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
        manifest = [{'platform': 'windows-amd64', 'variant': variant,
                     'binarySHA256': hashlib.sha256(files['build/cores/windows-amd64/' + filename]).hexdigest()}
                    for variant, filename in [('modern', 'xray.exe'), ('legacy', 'xray-legacy.exe')]]
        (self.root / 'build/cores/manifest.json').write_text(json.dumps(manifest))

    def test_portable_rejects_tampered_legacy_binary(self):
        (self.root / 'build/cores/windows-amd64/xray-legacy.exe').write_bytes(b'tampered')
        with self.assertRaisesRegex(AssertionError, 'Legacy core binary checksum mismatch'):
            self.module.main()

    def test_installer_build_rejects_tampered_legacy_before_docker(self):
        import os
        import subprocess
        script = self.root / 'build/build-installer.sh'
        script.write_bytes((Path(__file__).parent / 'build-installer.sh').read_bytes())
        fake_bin = self.root / 'fake-bin'
        fake_bin.mkdir()
        sudo = fake_bin / 'sudo'
        sudo.write_text('#!/bin/sh\nprintf "Docker must not run\\n" >&2\nexit 99\n')
        sudo.chmod(0o755)
        (self.root / 'build/cores/windows-amd64/xray-legacy.exe').write_bytes(b'tampered')
        result = subprocess.run(['bash', str(script)], capture_output=True, text=True,
                                env={**os.environ, 'PATH': str(fake_bin) + os.pathsep + os.environ['PATH']})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Legacy Xray core checksum mismatch', result.stderr)
        self.assertNotIn('Docker must not run', result.stderr)

    def test_portable_contains_both_cores_and_licenses_linux_stays_management_only(self):
        import tarfile
        import zipfile
        with redirect_stdout(io.StringIO()):
            self.module.main()
        with zipfile.ZipFile(next(self.module.OUT.glob('*-windows-amd64.zip'))) as archive:
            for filename in ('xray.exe', 'xray-legacy.exe', 'XRAY-LICENSE.txt', 'XRAY-LEGACY-LICENSE.txt', 'manifest.json'):
                self.assertIn('core/' + filename, archive.namelist())
                self.assertEqual(archive.read('core/' + filename), (self.root / 'build/cores' / ('windows-amd64' if filename.endswith('.exe') else '') / filename).read_bytes())
            self.assertEqual(archive.read('THIRD-PARTY.md'), b'notices')
        with tarfile.open(next(self.module.OUT.glob('*.tar.gz'))) as archive:
            self.assertFalse(any('xray' in name.lower() or name.startswith('core/') for name in archive.getnames()))


if __name__ == '__main__':
    unittest.main()
