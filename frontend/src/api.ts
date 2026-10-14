import type {User} from './types';
let csrf='';
export function setCSRF(v:string){csrf=v}
export async function api<T>(path:string,options:RequestInit={}):Promise<T>{const headers=new Headers(options.headers);if(options.body)headers.set('Content-Type','application/json');if(options.method&&options.method!=='GET')headers.set('X-CSRF-Token',csrf);const res=await fetch('/api'+path,{...options,headers,credentials:'same-origin'});if(!res.ok){const body=await res.json().catch(()=>({error:'Ошибка запроса'}));throw new Error(body.error||'Ошибка запроса')}if(res.status===204)return undefined as T;return res.json()}
export async function restore(){const u=await api<User>('/auth/me');setCSRF(u.csrfToken);return u}
