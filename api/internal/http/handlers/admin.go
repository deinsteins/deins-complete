package handlers

import (
	"deinscomplete/api/internal/account"
	"deinscomplete/api/internal/accountauth"
	"deinscomplete/api/internal/http/middleware"
	"deinscomplete/api/internal/http/response"
	"deinscomplete/api/internal/usage"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type AdminHandler struct {
	repo                 *account.Repository
	monthly              usage.MonthlyTracker
	qualitySamplePercent int
}

func NewAdminHandler(repo *account.Repository, monthly usage.MonthlyTracker, qualitySamplePercent int) *AdminHandler {
	return &AdminHandler{repo: repo, monthly: monthly, qualitySamplePercent: qualitySamplePercent}
}
func AdminPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminHTML))
}
func (h *AdminHandler) Overview(w http.ResponseWriter, r *http.Request) {
	summary, err := h.repo.AdminSummary(r.Context())
	if err != nil {
		response.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
		return
	}
	response.WriteJSON(w, http.StatusOK, summary)
}
func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.AdminListUsers(r.Context(), r.URL.Query().Get("q"), 100)
	if err != nil {
		response.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
		return
	}
	type userRow struct {
		account.AdminUser
		MonthlyUsed int `json:"monthlyUsed"`
	}
	result := make([]userRow, 0, len(users))
	for _, user := range users {
		row := userRow{AdminUser: user}
		if h.monthly != nil {
			used, err := h.monthly.Usage(r.Context(), "user:"+user.ID)
			if err != nil {
				response.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
				return
			}
			row.MonthlyUsed = used
		}
		result = append(result, row)
	}
	response.WriteJSON(w, http.StatusOK, result)
}
func (h *AdminHandler) Installations(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.AdminListInstallations(r.Context(), r.URL.Query().Get("userId"), 100)
	if err != nil {
		response.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
		return
	}
	response.WriteJSON(w, http.StatusOK, items)
}
func (h *AdminHandler) Invites(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.AdminListInvites(r.Context(), 100)
	if err != nil {
		response.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
		return
	}
	response.WriteJSON(w, http.StatusOK, items)
}
func (h *AdminHandler) Quality(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	quality, err := h.repo.AdminQuality(r.Context(), days)
	if err != nil {
		response.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable.", middleware.GetRequestID(r.Context()))
		return
	}
	quality.SamplePercent = h.qualitySamplePercent
	response.WriteJSON(w, http.StatusOK, quality)
}
func (h *AdminHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var body struct {
		Email string `json:"email"`
		Days  int    `json:"days"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil || account.NormalizeEmail(body.Email) == "" {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invite request is invalid.", middleware.GetRequestID(r.Context()))
		return
	}
	if body.Days == 0 {
		body.Days = 7
	}
	if body.Days < 1 || body.Days > 30 {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invite expiry must be 1-30 days.", middleware.GetRequestID(r.Context()))
		return
	}
	code, err := accountauth.NewOpaqueToken()
	if err == nil {
		_, err = h.repo.CreateInvite(r.Context(), accountauth.HashToken(code), body.Email, time.Now().Add(time.Duration(body.Days)*24*time.Hour))
	}
	if err != nil {
		response.WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Invite could not be created.", middleware.GetRequestID(r.Context()))
		return
	}
	response.WriteJSON(w, http.StatusCreated, map[string]string{"email": account.NormalizeEmail(body.Email), "code": code, "expiresInDays": strconv.Itoa(body.Days)})
}
func (h *AdminHandler) SetPlan(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Plan string `json:"plan"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body) != nil || (body.Plan != "free" && body.Plan != "pro") {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Plan request is invalid.", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.repo.SetUserPlan(r.Context(), r.PathValue("id"), body.Plan); err != nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "User not found.", middleware.GetRequestID(r.Context()))
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
func (h *AdminHandler) RevokeInstallation(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.AdminRevokeInstallation(r.Context(), r.PathValue("id")); err != nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Installation not found.", middleware.GetRequestID(r.Context()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

const adminHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>DeinsComplete Admin</title>
<style>
:root{color-scheme:dark;--bg:#0f1115;--panel:#171a21;--line:#2a2f3a;--text:#eef2f8;--muted:#99a3b3;--accent:#58c4a7;--warn:#f1b65c;--bad:#f17878}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font:14px/1.45 system-ui,-apple-system,Segoe UI,sans-serif}header{display:flex;gap:16px;align-items:center;justify-content:space-between;padding:18px 24px;border-bottom:1px solid var(--line);background:#12151b;position:sticky;top:0}h1{font-size:18px;margin:0}main{padding:24px;max-width:1280px;margin:auto}.auth,.grid,.section{background:var(--panel);border:1px solid var(--line);border-radius:8px}.auth{display:flex;gap:10px;padding:14px;margin-bottom:18px}.grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:1px;overflow:hidden;margin-bottom:18px}.metric{padding:16px;background:var(--panel)}.metric b{display:block;font-size:24px}.metric span,.muted{color:var(--muted)}.section{margin:18px 0;overflow:hidden}.section h2{font-size:15px;margin:0;padding:14px 16px;border-bottom:1px solid var(--line)}.toolbar{display:flex;gap:10px;align-items:center;padding:12px 16px;border-bottom:1px solid var(--line)}.notice{padding:10px 16px;color:var(--muted);border-bottom:1px solid var(--line)}.notice.warn{color:var(--warn);background:#211c13}input,select,button{height:34px;border-radius:6px;border:1px solid var(--line);background:#0f1218;color:var(--text);padding:0 10px}button{cursor:pointer;background:#1d2430}button.primary{background:var(--accent);color:#07110e;border-color:transparent;font-weight:650}button.warn{color:var(--warn)}button.bad{color:var(--bad)}table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:10px 12px;border-top:1px solid var(--line);vertical-align:middle}th{color:var(--muted);font-weight:600;background:#141820}.right{text-align:right}.code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.hidden{display:none}@media(max-width:800px){header,.auth,.toolbar{display:block}.auth input,.auth button,.toolbar input,.toolbar button{width:100%;margin:4px 0}.grid{grid-template-columns:1fr 1fr}main{padding:14px}td,th{white-space:nowrap}}
</style>
</head>
<body>
<header><h1>DeinsComplete Admin</h1><div id="state" class="muted">Locked</div></header>
<main>
<div class="auth"><input id="token" type="password" placeholder="Admin token" autocomplete="current-password"><button class="primary" onclick="saveToken()">Unlock</button><button onclick="logout()">Lock</button></div>
<div class="grid" id="summary"></div>
<section class="section"><h2>Completion quality <span class="muted">(privacy-safe aggregates)</span></h2><div class="toolbar"><label for="qualityDays" class="muted">Window</label><select id="qualityDays" onchange="loadQuality()"><option value="7">7 days</option><option value="30">30 days</option><option value="90">90 days</option></select></div><div id="qualityNotice" class="notice"></div><div class="grid" id="qualitySummary"></div><h2>Daily UTC trend</h2><table><thead><tr><th>Day</th><th>Shown</th><th>Accepted</th><th>Acceptance</th><th>p95 latency</th></tr></thead><tbody id="qualityTrend"></tbody></table><h2>Explicit negative feedback</h2><table><thead><tr><th>Reason category</th><th>Count</th></tr></thead><tbody id="qualityFeedback"></tbody></table><h2>Breakdown</h2><table><thead><tr><th>Dimension</th><th>Value</th><th>Shown</th><th>Accepted</th><th>Acceptance</th></tr></thead><tbody id="qualityDimensions"></tbody></table></section>
<section class="section"><h2>Users</h2><div class="toolbar"><input id="query" placeholder="Search email"><button onclick="loadAll()">Search</button><input id="inviteEmail" placeholder="Invite email"><input id="inviteDays" type="number" min="1" max="30" value="7"><button class="primary" onclick="createInvite()">Create invite</button></div><table><thead><tr><th>Email</th><th>Plan</th><th>Usage</th><th>Installations</th><th>Last seen</th><th class="right">Action</th></tr></thead><tbody id="users"></tbody></table></section>
<section class="section"><h2>Installations</h2><table><thead><tr><th>ID</th><th>User</th><th>Status</th><th>Last seen</th><th class="right">Action</th></tr></thead><tbody id="installations"></tbody></table></section>
<section class="section"><h2>Invites</h2><div class="toolbar hidden" id="newInvite"></div><table><thead><tr><th>Email</th><th>Expires</th><th>Used</th><th>Created</th></tr></thead><tbody id="invites"></tbody></table></section>
</main>
<script>
let token=sessionStorage.getItem("deinscomplete.adminToken")||"";document.getElementById("token").value=token;
const fmt=x=>x?new Date(x).toLocaleString():"-";
const esc=x=>String(x??"").replace(/[&<>"']/g,c=>({"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;"}[c]));
async function api(path,options={}){const r=await fetch(path,{...options,headers:{"Authorization":"Bearer "+token,"Content-Type":"application/json",...(options.headers||{})}});if(!r.ok)throw new Error((await r.json().catch(()=>({error:{message:r.statusText}}))).error.message);return r.status===204?null:r.json()}
function saveToken(){token=document.getElementById("token").value.trim();sessionStorage.setItem("deinscomplete.adminToken",token);loadAll()}
function logout(){sessionStorage.removeItem("deinscomplete.adminToken");token="";document.getElementById("state").textContent="Locked"}
async function loadAll(){try{document.getElementById("state").textContent="Loading";const [s,u,i,v,q]=await Promise.all([api("/v1/admin/overview"),api("/v1/admin/users?q="+encodeURIComponent(document.getElementById("query").value)),api("/v1/admin/installations"),api("/v1/admin/invites"),api("/v1/admin/quality?days="+document.getElementById("qualityDays").value)]);summary(s);users(u);installations(i);invites(v);quality(q);document.getElementById("state").textContent="Ready"}catch(e){document.getElementById("state").textContent=e.message}}
function summary(s){document.getElementById("summary").innerHTML=[["Users",s.Users],["Linked installations",s.LinkedInstallations],["Active installations",s.ActiveInstallations],["Pending invites",s.PendingInvites]].map(x=>"<div class=\"metric\"><span>"+x[0]+"</span><b>"+x[1]+"</b></div>").join("")}
async function loadQuality(){try{quality(await api("/v1/admin/quality?days="+document.getElementById("qualityDays").value))}catch(e){document.getElementById("state").textContent=e.message}}
function quality(q){const s=q.Summary;const notice=document.getElementById("qualityNotice");notice.className="notice"+(s.Shown<20?" warn":"");notice.textContent=s.Shown<20?"Directional only: fewer than 20 shown events in this window. Server sampling: "+q.SamplePercent+"%.":"Server sampling: "+q.SamplePercent+"%. Counts are sampled; acceptance ratios compare paired events.";document.getElementById("qualitySummary").innerHTML=[["Shown",s.Shown],["Accepted",s.Accepted],["Acceptance",Math.round(s.AcceptanceRate*100)+"%"],["p95 latency",Math.round(s.P95LatencyMS)+" ms"],["Helpful",s.Helpful],["Not helpful",s.NotHelpful]].map(x=>"<div class=\"metric\"><span>"+x[0]+"</span><b>"+x[1]+"</b></div>").join("");document.getElementById("qualityTrend").innerHTML=(q.Trend||[]).map(x=>"<tr><td class=\"code\">"+esc(x.Day)+"</td><td>"+x.Shown+"</td><td>"+x.Accepted+"</td><td>"+Math.round(x.AcceptanceRate*100)+"%</td><td>"+Math.round(x.P95LatencyMS)+" ms</td></tr>").join("");document.getElementById("qualityFeedback").innerHTML=(q.Feedback||[]).map(x=>"<tr><td class=\"code\">"+esc(x.Reason)+"</td><td>"+x.Count+"</td></tr>").join("")||"<tr><td colspan=\"2\" class=\"muted\">No explicit negative feedback in this window.</td></tr>";document.getElementById("qualityDimensions").innerHTML=(q.Dimensions||[]).map(x=>"<tr><td>"+esc(x.Kind)+"</td><td class=\"code\">"+esc(x.Value)+"</td><td>"+x.Shown+"</td><td>"+x.Accepted+"</td><td>"+Math.round(x.AcceptanceRate*100)+"%</td></tr>").join("")}
function users(rows){document.getElementById("users").innerHTML=rows.map(u=>"<tr><td>"+esc(u.Email)+"<div class=\"muted code\">"+esc(u.ID)+"</div></td><td><select onchange=\"setPlan('"+esc(u.ID)+"',this.value)\"><option "+(u.Plan==="free"?"selected":"")+">free</option><option "+(u.Plan==="pro"?"selected":"")+">pro</option></select></td><td>"+u.MonthlyUsed+"</td><td>"+u.Installations+"</td><td>"+fmt(u.LastSeenAt)+"</td><td class=\"right\"><button onclick=\"loadUserInstallations('"+esc(u.ID)+"')\">Devices</button></td></tr>").join("")}
function installations(rows){document.getElementById("installations").innerHTML=rows.map(i=>"<tr><td class=\"code\">"+esc(i.ID)+"</td><td>"+esc(i.Email||"-")+"</td><td>"+esc(i.Status)+"</td><td>"+fmt(i.LastSeenAt)+"</td><td class=\"right\">"+(i.Status==="active"?"<button class=\"bad\" onclick=\"revokeInstallation('"+esc(i.ID)+"')\">Revoke</button>":"")+"</td></tr>").join("")}
function invites(rows){document.getElementById("invites").innerHTML=rows.map(i=>"<tr><td>"+esc(i.Email||"-")+"</td><td>"+fmt(i.ExpiresAt)+"</td><td>"+fmt(i.UsedAt)+"</td><td>"+fmt(i.CreatedAt)+"</td></tr>").join("")}
async function createInvite(){const data=await api("/v1/admin/invites",{method:"POST",body:JSON.stringify({email:document.getElementById("inviteEmail").value,days:Number(document.getElementById("inviteDays").value)})});document.getElementById("newInvite").classList.remove("hidden");document.getElementById("newInvite").innerHTML="<span>Invite code for "+esc(data.email)+": <span class=\"code\">"+esc(data.code)+"</span></span>";loadAll()}
async function setPlan(id,plan){await api("/v1/admin/users/"+id+"/plan",{method:"POST",body:JSON.stringify({plan})});loadAll()}
async function loadUserInstallations(id){installations(await api("/v1/admin/installations?userId="+encodeURIComponent(id)))}
async function revokeInstallation(id){if(confirm("Revoke this installation?")){await api("/v1/admin/installations/"+id+"/revoke",{method:"POST"});loadAll()}}
if(token)loadAll();
</script>
</body>
</html>`
