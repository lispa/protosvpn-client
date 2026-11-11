import {render,screen,cleanup} from '@testing-library/react';
import {it,expect,vi,afterEach} from 'vitest';
import userEvent from '@testing-library/user-event';
import {I18nProvider} from './i18n';
import {WindowControls} from './WindowControls';

afterEach(()=>{cleanup();localStorage.clear();vi.unstubAllGlobals();});
it('does not show nonfunctional window controls in browser preview',()=>{
 render(<I18nProvider><WindowControls/></I18nProvider>);
 expect(screen.queryByRole('button')).toBeNull();
});
it('routes native controls to minimise, toggle maximise and quit',async()=>{
 const runtime={WindowMinimise:vi.fn(),WindowToggleMaximise:vi.fn(),Quit:vi.fn()};
 vi.stubGlobal('runtime',runtime);localStorage.setItem('protosvpn.language','en');
 render(<I18nProvider><WindowControls/></I18nProvider>);const user=userEvent.setup();
 await user.click(screen.getByRole('button',{name:'Minimise window'}));
 await user.click(screen.getByRole('button',{name:'Maximise / restore window'}));
 await user.click(screen.getByRole('button',{name:'Maximise / restore window'}));
 await user.click(screen.getByRole('button',{name:'Close application'}));
 expect(runtime.WindowMinimise).toHaveBeenCalledTimes(1);
 expect(runtime.WindowToggleMaximise).toHaveBeenCalledTimes(2);
 expect(runtime.Quit).toHaveBeenCalledTimes(1);
});
it('labels native controls in Russian',()=>{
 vi.stubGlobal('runtime',{});localStorage.setItem('protosvpn.language','ru');
 render(<I18nProvider><WindowControls/></I18nProvider>);
 expect(screen.getByRole('button',{name:'Свернуть окно'})).toBeInTheDocument();
 expect(screen.getByRole('button',{name:'Развернуть / восстановить окно'})).toBeInTheDocument();
 expect(screen.getByRole('button',{name:'Закрыть приложение'})).toBeInTheDocument();
});
