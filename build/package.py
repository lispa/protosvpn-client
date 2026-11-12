import hashlib,os,tarfile,zipfile,json
from pathlib import Path
ROOT=Path(__file__).resolve().parent.parent
OUT=ROOT/'release';VERSION='0.1.3-preview'
def main():
    OUT.mkdir(exist_ok=True)
    win=ROOT/'build/bin/ProtosVPN-windows-amd64.exe';linux=ROOT/'build/bin/ProtosVPN-linux-amd64'
    assert win.is_file() and linux.is_file(),'Build both production artifacts before packaging'
    assert win.read_bytes()[:2]==b'MZ' and linux.read_bytes()[:4]==b'\x7fELF','Invalid executable formats'
    readme=(ROOT/'README.md').read_bytes()
    wp=OUT/f'ProtosVPN-{VERSION}-windows-amd64.zip'
    with zipfile.ZipFile(wp,'w',zipfile.ZIP_DEFLATED) as z:
        z.write(win,'ProtosVPN.exe');z.writestr('README.ru.md',readme);z.write(ROOT/'VERIFICATION.md','VERIFICATION.md')
    lp=OUT/f'ProtosVPN-{VERSION}-ubuntu24-amd64.tar.gz'
    with tarfile.open(lp,'w:gz') as t:
        t.add(linux,arcname='ProtosVPN');t.add(ROOT/'README.md',arcname='README.ru.md');t.add(ROOT/'VERIFICATION.md',arcname='VERIFICATION.md')
    sp=OUT/f'ProtosVPN-{VERSION}-source.zip'
    with zipfile.ZipFile(sp,'w',zipfile.ZIP_DEFLATED) as z:
        for base,dirs,files in os.walk(ROOT):
            rel=Path(base).relative_to(ROOT)
            dirs[:]=[d for d in dirs if d not in ['node_modules','.git','release','coverage'] and not (rel==Path('build') and d in ['bin','qa'])]
            for file in files:
                p=Path(base)/file
                if file.startswith('.env') or p.suffix in ['.syso','.log','.ovpn','.key','.pem','.crt','.pfx','.p12','.tmp']:continue
                z.write(p,'protosvpn-client/'+str(p.relative_to(ROOT)))
    paths=[win,linux,wp,lp,sp]
    checks=''.join(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+str(p.relative_to(ROOT))+'\n' for p in paths)
    (OUT/'SHA256SUMS.txt').write_text(checks)
    print(json.dumps({'artifacts':[{'path':str(p),'bytes':p.stat().st_size,'sha256':hashlib.sha256(p.read_bytes()).hexdigest()} for p in paths]},indent=2))
if __name__=='__main__':main()
