package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"wechat-obsidian-bot/internal/license"
)

type Activation struct {
	ID         string `json:"id"`
	MachineID  string `json:"machine_id"`
	TxnID      string `json:"txn_id"`
	Status     string `json:"status"` // pending, approved, rejected
	LicenseKey string `json:"license_key,omitempty"`
	CreatedAt  string `json:"created_at"`
	ApprovedAt string `json:"approved_at,omitempty"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	Data struct {
		Activations []*Activation `json:"activations"`
	} `json:"data"`
}

func NewStore(path string) *Store {
	s := &Store{path: path}
	s.Data.Activations = []*Activation{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &s.Data)
	}
	return s
}

func (s *Store) save() {
	s.mu.RLock()
	data, _ := json.MarshalIndent(&s.Data, "", "  ")
	s.mu.RUnlock()
	os.WriteFile(s.path, data, 0644)
}

func (s *Store) Add(a *Activation) {
	s.mu.Lock()
	s.Data.Activations = append(s.Data.Activations, a)
	s.mu.Unlock()
	s.save()
}

func (s *Store) Approve(id string) *Activation {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.Data.Activations {
		if a.ID == id && a.Status == "pending" {
			a.Status = "approved"
			a.LicenseKey = license.GenerateKey(a.MachineID, "permanent")
			a.ApprovedAt = time.Now().Format("2006-01-02 15:04:05")
			s.save()
			return a
		}
	}
	return nil
}

func (s *Store) GetPending() []*Activation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var pending []*Activation
	for _, a := range s.Data.Activations {
		if a.Status == "pending" {
			pending = append(pending, a)
		}
	}
	return pending
}

func (s *Store) FindByID(id string) *Activation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.Data.Activations {
		if a.ID == id {
			return a
		}
	}
	return nil
}

// ---- HTTP Handlers ----

var store *Store

func Start(port string) error {
	store = NewStore("activations.json")

	http.HandleFunc("/api/activate", cors(handleActivate))
	http.HandleFunc("/api/approve", cors(handleApprove))
	http.HandleFunc("/api/key", cors(handleGetKey))
	http.HandleFunc("/api/pending", cors(handlePending))
	http.HandleFunc("/admin", handleAdminPage)
	http.Handle("/", http.FileServer(http.Dir("docs")))

	log.Printf("[server] 激活服务启动在 http://localhost:%s", port)
	log.Printf("[server] 管理面板: http://localhost:%s/admin", port)
	return http.ListenAndServe(":"+port, nil)
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			return
		}
		next(w, r)
	}
}

func handleActivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST only", 405)
		return
	}
	var req struct {
		MachineID string `json:"machine_id"`
		TxnID     string `json:"txn_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}
	if req.MachineID == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "设备码不能为空"})
		return
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	a := &Activation{
		ID:        id,
		MachineID: req.MachineID,
		TxnID:     req.TxnID,
		Status:    "pending",
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
	store.Add(a)

	log.Printf("[server] new activation: %s → %s", req.MachineID, id)
	json.NewEncoder(w).Encode(map[string]string{
		"id":     id,
		"status": "pending",
		"msg":    "提交成功，等待管理员验证付款后即可获取 Key",
	})
}

func handleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST only", 405)
		return
	}
	var req struct{ ID string `json:"id"` }
	json.NewDecoder(r.Body).Decode(&req)

	a := store.Approve(req.ID)
	if a == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "未找到该申请或已处理"})
		return
	}
	log.Printf("[server] approved: %s → %s...", a.MachineID, a.LicenseKey[:30])
	json.NewEncoder(w).Encode(a)
}

func handleGetKey(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	a := store.FindByID(id)
	if a == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "未找到"})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      a.Status,
		"license_key": a.LicenseKey,
		"machine_id":  a.MachineID,
	})
}

func handlePending(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(store.GetPending())
}

func handleAdminPage(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("admin").Parse(adminHTML))
	tmpl.Execute(w, nil)
}

const adminHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>激活管理 — WeChat Obsidian Bot</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#f5f5f5;color:#1a1a2e}
.container{max-width:900px;margin:0 auto;padding:30px 24px}
h1{margin-bottom:8px}
.subtitle{color:#888;margin-bottom:24px}
.card{background:white;border-radius:12px;padding:24px;box-shadow:0 2px 12px rgba(0,0,0,0.08);margin-bottom:20px}
table{width:100%;border-collapse:collapse}
th,td{padding:12px 16px;text-align:left;border-bottom:1px solid #e0e0e0}
th{background:#f8f9fc;font-weight:600}
.btn{padding:8px 20px;border-radius:6px;border:none;font-size:0.9em;font-weight:600;cursor:pointer}
.btn-approve{background:#4caf50;color:white}
.btn-approve:hover{background:#43a047}
.empty{text-align:center;color:#999;padding:40px}
.key-display{font-family:monospace;font-size:0.75em;word-break:break-all}
.badge{padding:3px 8px;border-radius:4px;font-size:0.8em}
.badge-pending{background:#fff3cd;color:#856404}
.badge-approved{background:#d4edda;color:#155724}
.refresh{float:right;color:#667eea;cursor:pointer;text-decoration:underline}
</style>
</head>
<body>
<div class="container">
<h1>🔐 激活管理 <span class="refresh" onclick="loadPending()">刷新</span></h1>
<p class="subtitle">待处理的激活请求 · 每10秒自动刷新</p>

<div class="card">
<table>
<thead><tr><th>时间</th><th>设备码</th><th>交易号</th><th>状态</th><th>License Key</th><th>操作</th></tr></thead>
<tbody id="pendingList"><tr><td colspan="6" class="empty">加载中...</td></tr></tbody>
</table>
</div>

<div id="result" style="display:none" class="card"></div>
</div>

<script>
async function loadPending() {
  try {
    const resp = await fetch('/api/pending');
    const list = await resp.json();
    const tbody = document.getElementById('pendingList');
    if (list.length === 0) {
      tbody.innerHTML = '<tr><td colspan="6" class="empty">✅ 没有待处理的请求</td></tr>';
      return;
    }
    tbody.innerHTML = list.map(a =>
      '<tr>' +
      '<td>' + a.created_at + '</td>' +
      '<td><code>' + a.machine_id + '</code></td>' +
      '<td>' + (a.txn_id || '-') + '</td>' +
      '<td><span class="badge badge-pending">待审批</span></td>' +
      '<td>-</td>' +
      '<td><button class="btn btn-approve" onclick="approve(\'' + a.id + '\')">✅ 批准</button></td>' +
      '</tr>'
    ).join('');
  } catch(e) {
    document.getElementById('pendingList').innerHTML = '<tr><td colspan="6" class="empty">加载失败</td></tr>';
  }
}

async function approve(id) {
  if (!confirm('确认已收到付款？将自动生成永久 License Key。')) return;
  try {
    const resp = await fetch('/api/approve', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({id})
    });
    const data = await resp.json();
    if (data.error) { alert(data.error); return; }

    document.getElementById('result').style.display = 'block';
    document.getElementById('result').innerHTML =
      '<strong>✅ 已批准！</strong><br><br>' +
      '设备码: <code>' + data.machine_id + '</code><br>' +
      'License Key:<br><div class="key-display" style="padding:12px;background:#e8f5e9;border-radius:6px;margin:8px 0;user-select:all">' + data.license_key + '</div>' +
      '<button class="btn btn-approve" onclick="navigator.clipboard.writeText(\'' + data.license_key + '\');this.textContent=\'✅ 已复制\'">📋 复制 Key</button>';

    loadPending();
  } catch(e) {
    alert('操作失败: ' + e);
  }
}

loadPending();
setInterval(loadPending, 10000);
</script>
</body>
</html>`
