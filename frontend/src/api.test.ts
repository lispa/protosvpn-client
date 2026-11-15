import {afterEach,describe,expect,it,vi} from 'vitest';
import {ApiError,apiErrorMessages,request} from './api';
import {translate} from './i18n';
afterEach(()=>vi.unstubAllGlobals());
describe('safe API errors',()=>{
  it.each(['ENGINE_START_FAILED','EGRESS_CHECK_FAILED','HELPER_START_FAILED','HELPER_EXITED','HELPER_TIMEOUT'])('shows safe diagnostic stage %s from actual API shape',async code=>{
    vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify({error:{code,message:'SECRET upstream'}}),{status:502})));
    const error=await request('/connect').catch((cause:unknown)=>cause);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({code});
    expect((error as ApiError).message).toContain(`[${code}]`);
    expect(translate('en',(error as ApiError).message)).not.toBe((error as ApiError).message);
    expect((error as ApiError).message).not.toContain('SECRET');
  });
  it.each(['CONNECT_FAILED','PROXY_FAILED','RESTORE_FAILED','UNSUPPORTED_PROFILE'] as const)('reads actual Go API nested error %s without exposing message',async code=>{
    vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify({error:{code,message:'SECRET backend output'}}),{status:502})));
    await expect(request('/connect')).rejects.toMatchObject({code,message:apiErrorMessages[code]});
  });
  it.each(['STORE_LOAD_FAILED','STORE_SAVE_FAILED','ENGINE_UNAVAILABLE','UNSUPPORTED_PROFILE','CONNECT_FAILED','PROXY_FAILED','RESTORE_FAILED','BUSY'] as const)('maps %s to allowlisted bilingual text',async code=>{
    vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify({code,message:'SECRET backend output'}),{status:500})));
    const error=await request('/connect').catch((cause:unknown)=>cause);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({code,message:apiErrorMessages[code]});
    expect(translate('en',(error as ApiError).message)).not.toBe((error as ApiError).message);
    expect((error as ApiError).message).not.toContain('SECRET');
  });
  it.each([{code:'SECRET'}, {errorCode:'SECRET'}, {code:'toString'}, {code:{secret:'value'}}, {error:'SECRET'},'SECRET'])('rejects unsafe error payload %j',async payload=>{
    vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify(payload),{status:500})));
    await expect(request('/connect')).rejects.toMatchObject({code:'REQUEST_FAILED',message:apiErrorMessages.REQUEST_FAILED});
  });
  it('accepts errorCode but not raw non-JSON error pages',async()=>{
    vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify({errorCode:'BUSY'}),{status:409})));
    await expect(request('/connect')).rejects.toMatchObject({code:'BUSY'});
    vi.stubGlobal('fetch',vi.fn(async()=>new Response('<html>SECRET</html>',{status:500})));
    await expect(request('/connect')).rejects.toMatchObject({code:'REQUEST_FAILED',message:apiErrorMessages.REQUEST_FAILED});
  });
});
