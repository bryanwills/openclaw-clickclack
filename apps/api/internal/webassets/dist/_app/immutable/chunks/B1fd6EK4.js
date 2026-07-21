import{n as e,r as t,t as n}from"./BEqsQr3A.js";var r=[{id:`bot:read`,label:`Read`,hint:`View channels, messages, and threads. No write access.`},{id:`bot:write`,label:`Read & write`,hint:`Post and edit messages, send DMs, upload attachments, and publish command menus.`},{id:`bot:admin`,label:`Admin`,hint:`Read & write, publish command menus, and manage channels. Use sparingly.`}],i={"bot:read":[`workspaces:read`,`channels:read`,`messages:read`,`threads:read`,`dms:read`,`realtime:read`,`profile:read`],"bot:write":[`workspaces:read`,`channels:read`,`messages:read`,`messages:write`,`threads:read`,`threads:write`,`dms:read`,`dms:write`,`realtime:read`,`uploads:write`,`profile:read`,`commands:write`],"bot:admin":[`workspaces:read`,`channels:read`,`channels:write`,`messages:read`,`messages:write`,`threads:read`,`threads:write`,`dms:read`,`dms:write`,`realtime:read`,`uploads:write`,`profile:read`,`commands:write`]};function a(e){let t=new Set;for(let n of e){t.add(n);let e=i[n];if(e)for(let n of e)t.add(n)}for(let n of[`bot:admin`,`bot:write`,`bot:read`]){let a=i[n];if(!a.every(e=>t.has(e)))continue;let o=new Set(a);return{bundle:n,bundleLabel:r.find(e=>e.id===n)?.label??n,extras:e.filter(e=>{if(e===n)return!1;let t=i[e];return t?!t.every(e=>o.has(e)):!o.has(e)})}}return{bundle:null,bundleLabel:null,extras:e}}async function o(t){return(await e(`/api/workspaces/${t}/bots`)).bots??[]}async function s(t,n){return e(`/api/workspaces/${t}/bots`,{method:`POST`,body:JSON.stringify(n)})}async function c(t,n){return(await e(`/api/workspaces/${t}/bots/${n}/tokens`)).bot_tokens??[]}async function l(t,n,r){return(await e(`/api/workspaces/${t}/bots/${n}/tokens`,{method:`POST`,body:JSON.stringify(r)})).bot_token}async function u(t,n,r){return(await e(`/api/workspaces/${t}/bots/${n}/setup-codes`,{method:`POST`,body:JSON.stringify(r)})).setup_code}async function d(t){return(await e(`/api/bot-tokens/${t}/revoke`,{method:`POST`,body:JSON.stringify({})})).bot_token}async function f(t,n){await e(`/api/workspaces/${t}/bots/${n}/membership`,{method:`DELETE`})}async function p(t){return(await e(`/api/bots/${t}`,{method:`DELETE`})).deleted_bot}async function m(){return(await e(`/api/me/bots`)).bots??[]}function h(e){if(e instanceof n){if(e.status===401)return`Sign in to manage bots.`;if(e.status===403)return`You don't have permission to manage bots in this workspace.`;if(e.status===404)return`That bot or workspace is no longer available.`;if(e.status===409)return`That handle is already taken. Try another.`;if(e.status===400)return e.message||`That request is invalid.`}return e instanceof Error?e.message:`Something went wrong`}function g(e){return!e.owner_user_id}function _(e){return e?e.filter(e=>!e.revoked_at):[]}function v(e){return e.toLowerCase().replace(/[^a-z0-9]+/g,`-`).replace(/^-+|-+$/g,``).slice(0,32)}function y(e){return e.slug.trim()||e.id}function b(e){return JSON.stringify(e)}function x(e){let t=e.replace(/^@/,``).toUpperCase().replace(/[^A-Z0-9]+/g,`_`).replace(/^_+|_+$/g,``);return t?`CLICKCLACK_${t}_BOT_TOKEN`:`CLICKCLACK_BOT_TOKEN`}function S(e){return`'${e.replaceAll(`'`,`'"'"'`)}'`}function C(e){let n=(e.baseURL||t()||(typeof window<`u`?window.location.origin:``)).replace(/\/$/,``),r=e.botHandle.replace(/^@/,``),i=e.mode===`single`?`CLICKCLACK_BOT_TOKEN`:x(r),a=n||`https://your-clickclack.example.com`,o=e.defaultTo?.trim()||`channel:general`,s=t=>{let n=[`workspace: ${b(e.workspace)},`,`botUserId: ${b(e.botUserID)},`,`defaultTo: ${b(o)},`];return e.allowFrom&&e.allowFrom.length>0&&!e.allowFrom.includes(`*`)&&n.push(`allowFrom: [${e.allowFrom.map(b).join(`, `)}],`),e.agentActivity&&n.push(`agentActivity: true,`),n.map(e=>t+e).join(`
`)};return e.mode===`named`?`{
  channels: {
    clickclack: {
      enabled: true,
      baseUrl: ${b(a)},
      defaultAccount: ${b(r)},
      accounts: {
        ${b(r)}: {
          token: { source: "env", provider: "default", id: ${b(i)} },
${s(`          `)}
        },
      },
    },
  },
}`:`{
  channels: {
    clickclack: {
      enabled: true,
      baseUrl: ${b(a)},
      token: { source: "env", provider: "default", id: ${b(i)} },
${s(`      `)}
    },
  },
}`}function w(e){let n=e.frontendURL||(typeof window<`u`?window.location.origin:``),r=(e.apiBaseURL||t()).replace(/\/$/,``),i=(e.baseURL||r||n).replace(/\/$/,``),a=(e.claimURL||``).replace(/\/$/,``),o=a&&r&&r!==n?a:i||`https://your-clickclack.example.com/`,s=e.botHandle.replace(/^@/,``),c=e.mode===`named`?` --account ${S(s)}`:``,l=o.endsWith(`/`)?`#`:`/#`;return`openclaw channels add clickclack${c} --code ${S(a&&o===a?`${o}#${e.code}`:`${o}${l}${e.code}`)}`}function T(e){let n=(e.baseURL||t()||(typeof window<`u`?window.location.origin:``)).replace(/\/$/,``)||`https://your-clickclack.example.com`,r=e.botHandle.replace(/^@/,``);return`openclaw channels add clickclack${e.mode===`named`?` \\\n  --account ${S(r)}`:``} \\
  --base-url ${S(n)} \\
  --token ${S(e.token)} \\
  --workspace ${S(e.workspace)}`}export{d as _,C as a,u as c,g as d,m as f,f as g,y as h,w as i,l,o as m,_ as n,T as o,c as p,h as r,s,r as t,p as u,v,a as y};