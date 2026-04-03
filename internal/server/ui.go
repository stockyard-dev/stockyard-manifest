package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashHTML))
}

const dashHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Manifest</title>
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#c45d2c;--rl:#e8753a;--leather:#a0845c;--ll:#c4a87a;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c44040;--blue:#4a7ec4;--mono:'JetBrains Mono',Consolas,monospace;--serif:'Libre Baskerville',Georgia,serif}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--mono);font-size:13px;line-height:1.6}
a{color:var(--rl);text-decoration:none}a:hover{color:var(--gold)}
.hdr{padding:.6rem 1.2rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}
.hdr h1{font-family:var(--serif);font-size:1rem}.hdr h1 span{color:var(--rl)}
.hdr-right{display:flex;gap:.7rem;align-items:center;font-size:.7rem}
.main{max-width:960px;margin:0 auto;padding:1rem 1.2rem}
.btn{font-family:var(--mono);font-size:.68rem;padding:.3rem .6rem;border:1px solid;cursor:pointer;background:transparent;transition:.15s;white-space:nowrap}
.btn-p{border-color:var(--rust);color:var(--rl)}.btn-p:hover{background:var(--rust);color:var(--cream)}
.btn-d{border-color:var(--bg3);color:var(--cm)}.btn-d:hover{border-color:var(--red);color:var(--red)}
.overview{display:flex;gap:1.5rem;margin-bottom:1rem;font-size:.7rem;color:var(--leather);flex-wrap:wrap}
.overview .stat b{display:block;font-size:1.2rem;color:var(--cream)}
.tabs{display:flex;gap:0;margin-bottom:1rem;border-bottom:1px solid var(--bg3)}
.tab{padding:.4rem 1rem;cursor:pointer;font-size:.75rem;color:var(--cm);border-bottom:2px solid transparent;transition:.15s}
.tab:hover{color:var(--cream)}.tab.active{color:var(--rl);border-bottom-color:var(--rl)}
.proj-card{background:var(--bg2);border:1px solid var(--bg3);padding:.6rem;margin-bottom:.4rem;cursor:pointer;transition:.1s}
.proj-card:hover{background:var(--bg3)}
.proj-card h3{font-size:.8rem;margin-bottom:.15rem}.proj-meta{font-size:.65rem;color:var(--cm);display:flex;gap:.7rem}
.dep-row{display:flex;align-items:center;gap:.5rem;padding:.35rem .5rem;border-bottom:1px solid var(--bg3);font-size:.72rem}
.dep-name{font-weight:600;width:180px;flex-shrink:0}.dep-ver{width:80px;color:var(--cd)}.dep-latest{width:80px}.dep-license{width:70px;color:var(--leather);font-size:.65rem}.dep-eco{font-size:.6rem;padding:.05rem .25rem;background:var(--bg3);color:var(--ll);border-radius:2px}
.outdated{color:var(--gold)}.current{color:var(--green)}
.vuln-row{display:flex;align-items:center;gap:.5rem;padding:.35rem .5rem;border-bottom:1px solid var(--bg3);font-size:.72rem}
.sev{font-size:.6rem;padding:.1rem .3rem;border:1px solid;border-radius:2px;font-weight:600;width:55px;text-align:center}
.sev-critical{border-color:var(--red);color:var(--red);background:rgba(196,64,64,.1)}.sev-high{border-color:var(--rl);color:var(--rl)}.sev-medium{border-color:var(--gold);color:var(--gold)}.sev-low{border-color:var(--cm);color:var(--cm)}
.lic-row{display:flex;align-items:center;gap:.5rem;padding:.3rem .5rem;border-bottom:1px solid var(--bg3);font-size:.75rem}
.lic-bar{height:4px;background:var(--bg3);flex:1;border-radius:2px;overflow:hidden}.lic-fill{height:100%;background:var(--rl)}
.empty{text-align:center;padding:2rem;color:var(--cm);font-style:italic;font-family:var(--serif)}
.modal-bg{position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,.65);display:flex;align-items:center;justify-content:center;z-index:100}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:95%;max-width:600px;max-height:90vh;overflow-y:auto}
.modal h2{font-family:var(--serif);font-size:.95rem;margin-bottom:1rem}
label.fl{display:block;font-size:.65rem;color:var(--leather);text-transform:uppercase;letter-spacing:1px;margin-bottom:.2rem;margin-top:.5rem}
input[type=text],textarea,select{background:var(--bg);border:1px solid var(--bg3);color:var(--cream);padding:.35rem .5rem;font-family:var(--mono);font-size:.78rem;width:100%;outline:none}
textarea{resize:vertical;min-height:80px}
.form-row{display:flex;gap:.5rem}.form-row>*{flex:1}
</style>
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital@0;1&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
</head><body>
<div class="hdr">
<h1><span>Manifest</span></h1>
<div class="hdr-right">
<select id="projSelect" onchange="switchProject(this.value)" style="background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.72rem;padding:.25rem .4rem"></select>
<button class="btn btn-p" onclick="showNewProject()">+ Project</button>
</div>
</div>
<div class="main"><div id="upgrade-banner" style="display:none;background:#241e18;border:1px solid #8b3d1a;border-left:3px solid #c45d2c;padding:.6rem 1rem;font-size:.78rem;color:#bfb5a3;margin-bottom:.8rem"><strong style="color:#f0e6d3">Free tier</strong> — 10 items max. <a href="https://stockyard.dev/manifest/" target="_blank" style="color:#e8753a">Upgrade to Pro →</a></div>
<div class="overview" id="overview"></div>
<div class="tabs">
<div class="tab active" data-tab="deps" onclick="switchTab('deps')">Dependencies</div>
<div class="tab" data-tab="vulns" onclick="switchTab('vulns')">Vulnerabilities</div>
<div class="tab" data-tab="licenses" onclick="switchTab('licenses')">Licenses</div>
</div>
<div id="pane-deps">
<div style="display:flex;justify-content:space-between;margin-bottom:.5rem">
<span style="font-size:.7rem;color:var(--leather)" id="depCount">-</span>
<div style="display:flex;gap:.3rem">
<button class="btn btn-p" onclick="showAddDep()">+ Dependency</button>
<button class="btn btn-p" onclick="showImport()">Import</button>
<button class="btn btn-d" onclick="exportSBOM()">SBOM Export</button>
</div>
</div>
<div id="depList"></div>
</div>
<div id="pane-vulns" style="display:none"><div id="vulnList"></div></div>
<div id="pane-licenses" style="display:none"><div id="licList"></div></div>
</div>
<div id="modal"></div>

<script>
let projects=[],curProject='',deps=[],vulns=[],licenses=[];
async function api(u,o){return(await fetch(u,o)).json()}
function esc(s){return String(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')}
function timeAgo(d){if(!d)return'never';const s=Math.floor((Date.now()-new Date(d))/1e3);if(s<60)return s+'s ago';if(s<3600)return Math.floor(s/60)+'m ago';return Math.floor(s/3600)+'h ago'}

async function init(){
  const[pd,sd]=await Promise.all([api('/api/projects'),api('/api/stats')]);
  projects=pd.projects||[];
  document.getElementById('projSelect').innerHTML='<option value="">Select project</option>'+projects.map(p=>'<option value="'+p.id+'">'+esc(p.name)+' ('+p.dep_count+' deps)</option>').join('');
  if(curProject)document.getElementById('projSelect').value=curProject;
  document.getElementById('overview').innerHTML=
    '<div class="stat"><b>'+sd.dependencies+'</b>Dependencies</div>'+
    '<div class="stat"><b style="color:'+(sd.vulnerabilities?'var(--red)':'var(--green)')+'">'+sd.vulnerabilities+'</b>Vulnerabilities</div>'+
    '<div class="stat"><b style="color:'+(sd.outdated?'var(--gold)':'var(--green)')+'">'+sd.outdated+'</b>Outdated</div>'+
    '<div class="stat"><b>'+sd.projects+'</b>Projects</div>';
  if(curProject)loadDeps();
}
function switchProject(id){curProject=id;loadDeps();loadVulns();loadLicenses()}
async function loadDeps(){
  if(!curProject){document.getElementById('depList').innerHTML='<div class="empty">Select a project.</div>';return}
  const d=await api('/api/projects/'+curProject+'/dependencies');deps=d.dependencies||[];
  document.getElementById('depCount').textContent=deps.length+' dependencies';
  document.getElementById('depList').innerHTML=deps.length?deps.map(dep=>
    '<div class="dep-row"><span class="dep-name">'+esc(dep.name)+(dep.vuln_count?' <span style="color:var(--red);font-size:.6rem">'+dep.vuln_count+' vuln</span>':'')+'</span>'+
    '<span class="dep-ver">'+esc(dep.version)+'</span>'+
    '<span class="dep-latest '+(dep.outdated?'outdated':'current')+'">'+esc(dep.latest_version||dep.version)+'</span>'+
    '<span class="dep-license">'+esc(dep.license||'-')+'</span>'+
    (dep.ecosystem?'<span class="dep-eco">'+esc(dep.ecosystem)+'</span>':'')+
    (dep.outdated?'<span style="color:var(--gold);font-size:.6rem">outdated</span>':'')+
    (dep.deprecated?'<span style="color:var(--red);font-size:.6rem">deprecated</span>':'')+
    '<span style="cursor:pointer;font-size:.55rem;color:var(--cm)" onclick="delDep(\''+dep.id+'\')">del</span></div>'
  ).join(''):'<div class="empty">No dependencies tracked yet.</div>'
}
async function loadVulns(){
  if(!curProject)return;
  const d=await api('/api/projects/'+curProject+'/vulnerabilities');vulns=d.vulnerabilities||[];
  document.getElementById('vulnList').innerHTML=vulns.length?vulns.map(v=>
    '<div class="vuln-row"><span class="sev sev-'+v.severity+'">'+v.severity+'</span><span style="flex:1;font-weight:600">'+esc(v.title)+'</span>'+
    (v.cve_id?'<span style="color:var(--leather);font-size:.65rem">'+esc(v.cve_id)+'</span>':'')+
    (v.fix_version?'<span style="color:var(--green);font-size:.65rem">fix: '+esc(v.fix_version)+'</span>':'')+
    '</div>').join(''):'<div class="empty">No vulnerabilities found.</div>'
}
async function loadLicenses(){
  if(!curProject)return;
  const d=await api('/api/projects/'+curProject+'/licenses');licenses=d.licenses||[];
  const max=Math.max(1,...licenses.map(l=>l.count));
  document.getElementById('licList').innerHTML=licenses.length?licenses.map(l=>
    '<div class="lic-row"><span style="width:120px;font-weight:600">'+esc(l.license)+'</span><div class="lic-bar"><div class="lic-fill" style="width:'+Math.round(l.count/max*100)+'%"></div></div><span style="width:30px;text-align:right;color:var(--cm)">'+l.count+'</span></div>'
  ).join(''):'<div class="empty">No license data.</div>'
}
function switchTab(t){
  document.querySelectorAll('.tab').forEach(el=>el.classList.toggle('active',el.dataset.tab===t));
  document.getElementById('pane-deps').style.display=t==='deps'?'':'none';
  document.getElementById('pane-vulns').style.display=t==='vulns'?'':'none';
  document.getElementById('pane-licenses').style.display=t==='licenses'?'':'none';
  if(t==='vulns')loadVulns();if(t==='licenses')loadLicenses()
}
async function delDep(id){await api('/api/dependencies/'+id,{method:'DELETE'});loadDeps();init()}

function showNewProject(){
  document.getElementById('modal').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal"><h2>New Project</h2><label class="fl">Name</label><input type="text" id="np-name"><label class="fl">Ecosystem</label><select id="np-eco"><option value="">Select</option><option>npm</option><option>go</option><option>pip</option><option>cargo</option><option>maven</option></select><label class="fl">Repo URL</label><input type="text" id="np-repo" placeholder="https://github.com/..."><div style="display:flex;gap:.5rem;margin-top:1rem"><button class="btn btn-p" onclick="saveProject()">Create</button><button class="btn btn-d" onclick="closeModal()">Cancel</button></div></div></div>'
}
async function saveProject(){const b={name:document.getElementById('np-name').value,ecosystem:document.getElementById('np-eco').value,repo_url:document.getElementById('np-repo').value};if(!b.name){alert('Name required');return};const r=await api('/api/projects',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)});curProject=r.id;closeModal();init()}

function showAddDep(){
  if(!curProject){alert('Select a project first');return}
  document.getElementById('modal').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal"><h2>Add Dependency</h2><label class="fl">Package Name</label><input type="text" id="ad-name" placeholder="express"><div class="form-row"><div><label class="fl">Version</label><input type="text" id="ad-ver" placeholder="4.18.2"></div><div><label class="fl">Latest Version</label><input type="text" id="ad-latest" placeholder="4.19.0"></div></div><div class="form-row"><div><label class="fl">License</label><input type="text" id="ad-lic" placeholder="MIT"></div><div><label class="fl">Ecosystem</label><input type="text" id="ad-eco" placeholder="npm"></div></div><div style="display:flex;gap:.5rem;margin-top:1rem"><button class="btn btn-p" onclick="saveDep()">Add</button><button class="btn btn-d" onclick="closeModal()">Cancel</button></div></div></div>'
}
async function saveDep(){const b={name:document.getElementById('ad-name').value,version:document.getElementById('ad-ver').value,latest_version:document.getElementById('ad-latest').value,license:document.getElementById('ad-lic').value,ecosystem:document.getElementById('ad-eco').value,direct:true};if(!b.name){alert('Name required');return};await api('/api/projects/'+curProject+'/dependencies',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)});closeModal();loadDeps();init()}

function showImport(){
  if(!curProject){alert('Select a project first');return}
  document.getElementById('modal').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal"><h2>Import Dependencies</h2><label class="fl">JSON Array</label><textarea id="im-data" rows="8" placeholder=\'[{"name":"express","version":"4.18.2","license":"MIT","ecosystem":"npm"}]\'></textarea><div style="display:flex;gap:.5rem;margin-top:1rem"><button class="btn btn-p" onclick="doImport()">Import</button><button class="btn btn-d" onclick="closeModal()">Cancel</button></div></div></div>'
}
async function doImport(){let deps;try{deps=JSON.parse(document.getElementById('im-data').value)}catch(e){alert('Invalid JSON');return};const r=await api('/api/projects/'+curProject+'/import',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({dependencies:deps})});alert('Imported '+r.imported+' dependencies');closeModal();loadDeps();init()}

function exportSBOM(){if(!curProject)return;window.open('/api/projects/'+curProject+'/sbom','_blank')}

function closeModal(){document.getElementById('modal').innerHTML=''}
init()
fetch('/api/tier').then(r=>r.json()).then(j=>{if(j.tier==='free'){var b=document.getElementById('upgrade-banner');if(b)b.style.display='block'}}).catch(()=>{var b=document.getElementById('upgrade-banner');if(b)b.style.display='block'});
</script></body></html>`
