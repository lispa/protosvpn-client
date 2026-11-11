import json,urllib.request,urllib.error
BASE='http://127.0.0.1:18761'
def req(path,body=None,method='GET',headers=None):
    h={} if headers is None else dict(headers)
    if body is not None:h['Content-Type']='application/json'
    r=urllib.request.Request(BASE+path,data=None if body is None else json.dumps(body).encode(),headers=h,method=method)
    try:
        with urllib.request.urlopen(r,timeout=10) as v:return v.status,json.load(v)
    except urllib.error.HTTPError as e:return e.code,json.load(e)
def main():
    code,state=req('/api/state');assert code==200
    token={'X-ProtosVPN-Token':state['csrfToken']};assert state['profiles']==[], 'Use a fresh owned preview process, not an existing user session'
    assert state['storageMode']=='memory' and state['connection']['engineAvailable'] is False
    dummy='11111111-1111-4111-8111-111111111111'
    uri=f'vless://{dummy}@198.51.100.10:443?security=tls&type=tcp#Helsinki%20TEST'
    code,_=req('/api/import',{'format':'vless','content':uri},'POST');assert code==403
    code,created=req('/api/import',{'format':'vless','content':uri},'POST',token);assert code==201
    p=created['profile'];assert p['name']=='Helsinki TEST' and p['protocol']=='vless' and p['port']==443
    assert dummy not in json.dumps(created) and 'content' not in p
    pid=p['id']
    code,_=req('/api/profiles/'+pid,{'name':'Helsinki renamed'},'PATCH',token);assert code==200
    code,r=req('/api/connect',{'id':pid},'POST',token);assert code==409 and r['error']['code']=='ENGINE_UNAVAILABLE'
    code,state=req('/api/state');assert state['connection']['status']=='disconnected' and state['profiles'][0]['name']=='Helsinki renamed'
    code,d=req('/api/diagnostics');assert code==200 and d['profileCount']==1 and dummy not in json.dumps(d) and 'host' not in d
    bad='client\ndev tun\nremote 198.51.100.11 1194\nup /tmp/DO-NOT-EXECUTE\n'
    code,result=req('/api/import',{'format':'openvpn','content':bad},'POST',token);assert code==400 and 'DO-NOT-EXECUTE' not in json.dumps(result)
    safe='client\ndev tun\nproto udp\nremote 198.51.100.11 1194\nauth-user-pass\n<ca>\nSYNTHETIC-OPAQUE-CERTIFICATE-NOT-REAL\n</ca>\n'
    code,r=req('/api/import',{'format':'openvpn','content':safe,'name':'OpenVPN TEST'},'POST',token);assert code==201 and r['profile']['requiresAuth'] is True
    oid=r['profile']['id'];assert 'SYNTHETIC' not in json.dumps(r)
    code,_=req('/api/select',{'id':oid},'POST',dict(token,Origin='https://untrusted.example'));assert code==403
    code,_=req('/api/select',{'id':oid},'POST',dict(token,Host='untrusted.example'));assert code==403
    code,_=req('/api/select',{'id':oid},'POST',token);assert code==200
    code,_=req('/api/profiles/'+oid,None,'DELETE',token);assert code==200
    code,_=req('/api/profiles/'+pid,None,'DELETE',token);assert code==200
    _,state=req('/api/state');assert state['profiles']==[] and not state['selectedId']
    print('PASS real Go HTTP: VLESS/OpenVPN import, rename/select/delete, secret redaction, unsafe config/Origin/Host/CSRF rejection, honest unavailable engine; test profiles removed')
if __name__=='__main__':main()
