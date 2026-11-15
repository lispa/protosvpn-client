"""Fetch pinned official Xray binaries; verify GitHub release SHA256 digests."""
from pathlib import Path
import hashlib
import json
import urllib.request
import zipfile
import io
import re

VERSION = 'v26.11.9'
ROOT = Path(__file__).resolve().parent.parent

def fetch(url):
    with urllib.request.urlopen(url, timeout=90) as response:
        return response.read()

def main():
    records = []
    for platform, archive, executable in [
        ('linux-amd64', 'Xray-linux-64.zip', 'xray'),
        ('windows-amd64', 'Xray-windows-64.zip', 'xray.exe'),
    ]:
        url = f'https://github.com/XTLS/Xray-core/releases/download/{VERSION}/{archive}'
        blob = fetch(url)
        digest = fetch(url + '.dgst').decode()
        sha = hashlib.sha256(blob).hexdigest()
        if sha.lower() not in re.findall(r'(?i)\b[0-9a-f]{64}\b', digest):
            raise RuntimeError('Official Xray SHA256 mismatch')
        z = zipfile.ZipFile(io.BytesIO(blob))
        if z.testzip():
            raise RuntimeError('Damaged Xray archive')
        folder = ROOT / 'build' / 'cores' / platform
        folder.mkdir(parents=True, exist_ok=True)
        member = next(n for n in z.namelist() if n == executable or n.endswith('/' + executable))
        output = folder / executable
        output.write_bytes(z.read(member))
        output.chmod(0o755)
        records.append({'platform': platform, 'version': VERSION, 'source': url,
                        'archiveSHA256': sha, 'binarySHA256': hashlib.sha256(output.read_bytes()).hexdigest()})
        print('Verified', platform, VERSION, sha)
    (ROOT / 'build' / 'cores' / 'manifest.json').write_text(json.dumps(records, indent=2) + '\n')
    license_blob = fetch(f'https://raw.githubusercontent.com/XTLS/Xray-core/{VERSION}/LICENSE')
    (ROOT / 'build' / 'cores' / 'XRAY-LICENSE.txt').write_bytes(license_blob)

if __name__ == '__main__':
    main()
