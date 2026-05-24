'use strict';

// ═══════════════════════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════════════════════

let token       = localStorage.getItem('token') || null;
let currentUser = JSON.parse(localStorage.getItem('user') || 'null');
let allTasks    = [];
let activeFilter   = 'all';
let priorityFilter = 'all';
let addFormOpen    = false;

// ── Timer state ────────────────────────────────────────────────────────────
// Stored in localStorage so timers survive page refresh.
// Shape: { [taskId]: { endTime: <unix ms>, taskTitle: string } }
let activeTimers = JSON.parse(localStorage.getItem('activeTimers') || '{}');

// ═══════════════════════════════════════════════════════════════════════════
// API Helper
// ═══════════════════════════════════════════════════════════════════════════

const BASE_URL = '/api/v1';

async function api(path, { method = 'GET', body } = {}) {
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(`${BASE_URL}${path}`, {
    method, headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || `Request failed (${res.status})`);
  return data.data;
}

// ═══════════════════════════════════════════════════════════════════════════
// Auth
// ═══════════════════════════════════════════════════════════════════════════

function saveAuth(t, u) {
  token = t; currentUser = u;
  localStorage.setItem('token', t);
  localStorage.setItem('user', JSON.stringify(u));
}

function clearAuth() {
  token = null; currentUser = null; allTasks = [];
  localStorage.removeItem('token'); localStorage.removeItem('user');
}

async function handleLogin(e) {
  e.preventDefault();
  const btn = document.getElementById('login-btn');
  const errEl = document.getElementById('login-error');
  setLoading(btn, true); showError(errEl, '');
  try {
    const r = await api('/auth/login', { method:'POST', body:{
      email: document.getElementById('login-email').value.trim(),
      password: document.getElementById('login-password').value,
    }});
    saveAuth(r.token, r.user); showApp();
  } catch(err) { showError(errEl, err.message); }
  finally { setLoading(btn, false); }
}

async function handleRegister(e) {
  e.preventDefault();
  const btn = document.getElementById('register-btn');
  const errEl = document.getElementById('register-error');
  setLoading(btn, true); showError(errEl, '');
  try {
    const r = await api('/auth/register', { method:'POST', body:{
      email: document.getElementById('reg-email').value.trim(),
      password: document.getElementById('reg-password').value,
    }});
    saveAuth(r.token, r.user); showApp();
  } catch(err) { showError(errEl, err.message); }
  finally { setLoading(btn, false); }
}

function handleLogout() {
  clearAuth(); showAuth(); toast('Signed out', 'success');
}

// ═══════════════════════════════════════════════════════════════════════════
// Add-Task Form Toggle  (the + button)
// ═══════════════════════════════════════════════════════════════════════════

function toggleAddForm() {
  addFormOpen = !addFormOpen;
  const btn   = document.getElementById('add-task-toggle');
  const panel = document.getElementById('add-task-panel');

  btn.classList.toggle('active', addFormOpen);
  btn.setAttribute('aria-expanded', addFormOpen);
  panel.classList.toggle('open', addFormOpen);
  panel.setAttribute('aria-hidden', !addFormOpen);

  if (addFormOpen) {
    // Focus the title input with a short delay (after animation starts)
    setTimeout(() => document.getElementById('task-title').focus(), 150);
  } else {
    // Reset form when closing
    document.getElementById('add-task-form').reset();
    document.getElementById('task-priority').value = 'medium';
    document.getElementById('task-recurring').value = 'none';
    document.getElementById('timer-enabled').checked = false;
    toggleTimerFields(false);
    showError(document.getElementById('add-task-error'), '');
  }
}

function toggleTimerFields(show) {
  document.getElementById('timer-duration-group').classList.toggle('hidden', !show);
}

// ═══════════════════════════════════════════════════════════════════════════
// Tasks — CRUD
// ═══════════════════════════════════════════════════════════════════════════

async function loadTasks() {
  showLoading(true);
  try {
    allTasks = await api('/tasks') || [];
    renderTasks();
    updateStats();
  } catch(err) {
    toast('Failed to load tasks: ' + err.message, 'error');
  } finally { showLoading(false); }
}

async function handleCreateTask(e) {
  e.preventDefault();
  const btn   = document.getElementById('add-task-btn');
  const errEl = document.getElementById('add-task-error');
  setLoading(btn, true); showError(errEl, '');

  // Build the request body
  const body = {
    title:       document.getElementById('task-title').value.trim(),
    description: document.getElementById('task-desc').value.trim(),
    priority:    document.getElementById('task-priority').value,
    recurring:   document.getElementById('task-recurring').value,
  };

  const dueDateVal = document.getElementById('task-due').value;
  if (dueDateVal) body.due_date = new Date(dueDateVal).toISOString();

  // Timer
  const timerEnabled = document.getElementById('timer-enabled').checked;
  if (timerEnabled) {
    const hours   = parseInt(document.getElementById('timer-hours').value,   10) || 0;
    const minutes = parseInt(document.getElementById('timer-minutes').value, 10) || 0;
    const totalMinutes = hours * 60 + minutes;
    if (totalMinutes <= 0) {
      showError(errEl, 'Timer duration must be greater than 0 minutes.');
      setLoading(btn, false);
      return;
    }
    body.timer_duration_minutes = totalMinutes;
  }

  try {
    const task = await api('/tasks', { method:'POST', body });
    allTasks.unshift(task);
    renderTasks();
    updateStats();
    toggleAddForm(); // close form
    toast('Task added! 🎉', 'success');
    // Auto-start timer if set
    if (task.timer_duration_minutes) {
      autoStartTimer(task);
    }
  } catch(err) { showError(errEl, err.message); }
  finally { setLoading(btn, false); }
}

async function toggleComplete(taskId, currentlyCompleted) {
  const checkbox = document.querySelector(`[data-check="${taskId}"]`);
  if (checkbox) checkbox.disabled = true;
  try {
    const updated = await api(`/tasks/${taskId}`, { method:'PUT', body:{ completed:!currentlyCompleted } });
    const idx = allTasks.findIndex(t => t.id === taskId);
    if (idx !== -1) allTasks[idx] = updated;
    renderTasks(); updateStats();
    // If recurring and just completed, show a hint
    if (updated.completed && updated.recurring !== 'none') {
      toast(`🔁 Recurring task completed! Next occurrence: ${nextRecurringDate(updated.recurring)}`, 'success');
    }
  } catch(err) {
    toast('Could not update: ' + err.message, 'error');
    if (checkbox) checkbox.checked = currentlyCompleted;
  } finally { if (checkbox) checkbox.disabled = false; }
}

async function handleUpdateTask(e) {
  e.preventDefault();
  const btn    = document.getElementById('edit-save-btn');
  const errEl  = document.getElementById('edit-error');
  const taskId = document.getElementById('edit-task-id').value;
  setLoading(btn, true); showError(errEl, '');

  const dueDateVal = document.getElementById('edit-due').value;
  const timerVal   = parseInt(document.getElementById('edit-timer').value, 10) || 0;

  try {
    const body = {
      title:       document.getElementById('edit-title').value.trim(),
      description: document.getElementById('edit-desc').value.trim(),
      priority:    document.getElementById('edit-priority').value,
      recurring:   document.getElementById('edit-recurring').value,
      timer_duration_minutes: timerVal > 0 ? timerVal : null,
    };
    if (dueDateVal) body.due_date = new Date(dueDateVal).toISOString();

    const updated = await api(`/tasks/${taskId}`, { method:'PUT', body });
    const idx = allTasks.findIndex(t => t.id === taskId);
    if (idx !== -1) allTasks[idx] = updated;
    renderTasks(); updateStats();
    closeEditModal();
    toast('Task updated ✓', 'success');
  } catch(err) { showError(errEl, err.message); }
  finally { setLoading(btn, false); }
}

async function deleteTask(taskId) {
  if (!confirm('Delete this task? This cannot be undone.')) return;
  try {
    await api(`/tasks/${taskId}`, { method:'DELETE' });
    allTasks = allTasks.filter(t => t.id !== taskId);
    // Stop any running timer for this task
    stopTimer(taskId);
    renderTasks(); updateStats();
    toast('Task deleted', 'success');
  } catch(err) { toast('Could not delete: ' + err.message, 'error'); }
}

// ═══════════════════════════════════════════════════════════════════════════
// ⏱ TIMER ENGINE
// ═══════════════════════════════════════════════════════════════════════════

/**
 * Request browser notification permission on first interaction.
 * WHY: Browsers require a user gesture before allowing notifications.
 */
function requestNotificationPermission() {
  if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission();
  }
}

/**
 * Start a countdown timer for a task.
 * @param {object} task - the full task object
 */
function startTimer(task) {
  requestNotificationPermission();
  const durationMs = task.timer_duration_minutes * 60 * 1000;
  activeTimers[task.id] = {
    endTime:   Date.now() + durationMs,
    taskTitle: task.title,
  };
  persistTimers();
  renderTasks(); // re-render to show countdown
  toast(`⏱ Timer started: ${formatDuration(task.timer_duration_minutes)}`, 'success');
}

/** Auto-start timer when a task is created with timer_duration_minutes set */
function autoStartTimer(task) {
  if (task.timer_duration_minutes && !activeTimers[task.id]) {
    startTimer(task);
  }
}

/** Stop (cancel) a running timer */
function stopTimer(taskId) {
  if (activeTimers[taskId]) {
    delete activeTimers[taskId];
    persistTimers();
    renderTasks();
    toast('Timer stopped', 'success');
  }
}

function persistTimers() {
  localStorage.setItem('activeTimers', JSON.stringify(activeTimers));
}

/**
 * Called every second by setInterval.
 * Checks all running timers and fires notifications for expired ones.
 */
function tickTimers() {
  const now = Date.now();
  let anyExpired = false;

  for (const [taskId, timer] of Object.entries(activeTimers)) {
    if (now >= timer.endTime) {
      delete activeTimers[taskId];
      anyExpired = true;
      fireTimerNotification(timer.taskTitle, taskId);
    }
  }

  if (anyExpired) {
    persistTimers();
    renderTasks();
  } else {
    // Just update countdown displays without full re-render (performance)
    updateAllCountdowns();
  }
}

/** Update all visible countdown badges in-place */
function updateAllCountdowns() {
  const now = Date.now();
  document.querySelectorAll('[data-timer-id]').forEach(el => {
    const taskId = el.dataset.timerId;
    const timer  = activeTimers[taskId];
    if (!timer) return;
    const remaining = timer.endTime - now;
    if (remaining > 0) {
      el.textContent = formatCountdown(remaining);
    }
  });
}

/**
 * Fire a browser notification + in-app alert when a timer expires.
 */
function fireTimerNotification(taskTitle, taskId) {
  // Browser notification (works even if tab is in background)
  if ('Notification' in window && Notification.permission === 'granted') {
    new Notification('⏰ Time\'s up! — Tasky', {
      body: `"${taskTitle}" timer has expired.`,
      tag:  `tasky-timer-${taskId}`,   // prevents duplicate notifications
      requireInteraction: true,        // stays visible until dismissed
    });
  }
  // In-app alert overlay
  document.getElementById('timer-alert-title').textContent = "⏰ Time's up!";
  document.getElementById('timer-alert-msg').textContent   = `"${taskTitle}" — your timer has expired.`;
  document.getElementById('timer-alert').classList.remove('hidden');
}

function dismissTimerAlert() {
  document.getElementById('timer-alert').classList.add('hidden');
}

// Format milliseconds → "1h 23m 45s"
function formatCountdown(ms) {
  const totalSec = Math.max(0, Math.floor(ms / 1000));
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  if (h > 0) return `${h}h ${pad(m)}m ${pad(s)}s`;
  if (m > 0) return `${m}m ${pad(s)}s`;
  return `${s}s`;
}

// Format total minutes → "2h 30m" or "45m"
function formatDuration(minutes) {
  if (!minutes) return '';
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  if (h > 0 && m > 0) return `${h}h ${m}m`;
  if (h > 0) return `${h}h`;
  return `${m}m`;
}

function pad(n) { return String(n).padStart(2, '0'); }

// Start the global timer tick (every second)
setInterval(tickTimers, 1000);

// ═══════════════════════════════════════════════════════════════════════════
// Recurring helpers
// ═══════════════════════════════════════════════════════════════════════════

const RECURRING_LABELS = { none:'', daily:'Daily', weekly:'Weekly', monthly:'Monthly' };
const RECURRING_ICONS  = { daily:'📅', weekly:'🗓️', monthly:'📆' };

function nextRecurringDate(recurring) {
  const d = new Date();
  if (recurring === 'daily')   d.setDate(d.getDate() + 1);
  if (recurring === 'weekly')  d.setDate(d.getDate() + 7);
  if (recurring === 'monthly') d.setMonth(d.getMonth() + 1);
  return d.toLocaleDateString('en-US', { month:'short', day:'numeric' });
}

// ═══════════════════════════════════════════════════════════════════════════
// Filtering
// ═══════════════════════════════════════════════════════════════════════════

function getFilteredTasks() {
  return allTasks.filter(t => {
    if (activeFilter   === 'active'    && t.completed)  return false;
    if (activeFilter   === 'completed' && !t.completed) return false;
    if (priorityFilter !== 'all' && t.priority !== priorityFilter) return false;
    return true;
  });
}

function setFilter(filter, btn) {
  activeFilter = filter;
  document.querySelectorAll('[data-filter]').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
  renderTasks();
}

function setPriorityFilter(priority, btn) {
  priorityFilter = priority;
  document.querySelectorAll('[data-priority]').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
  renderTasks();
}

// ═══════════════════════════════════════════════════════════════════════════
// Rendering
// ═══════════════════════════════════════════════════════════════════════════

function renderTasks() {
  const list      = document.getElementById('task-list');
  const emptyEl   = document.getElementById('empty-state');
  const filtered  = getFilteredTasks();

  list.innerHTML = '';

  if (filtered.length === 0) {
    emptyEl.classList.remove('hidden');
    return;
  }
  emptyEl.classList.add('hidden');
  filtered.forEach(task => list.appendChild(createTaskCard(task)));
}

function createTaskCard(task) {
  const card = document.createElement('div');
  const hasTimer = !!activeTimers[task.id];
  card.className = `task-card${task.completed ? ' completed' : ''}${hasTimer ? ' has-active-timer' : ''}`;
  card.setAttribute('role', 'listitem');

  const dueInfo  = task.due_date ? formatDueDate(task.due_date) : null;
  const overdue  = dueInfo?.overdue && !task.completed;
  const created  = new Date(task.created_at).toLocaleDateString('en-US', { month:'short', day:'numeric' });
  const timerData = activeTimers[task.id];
  const remainingMs = timerData ? timerData.endTime - Date.now() : 0;

  // Build badges row
  let badges = `<span class="badge priority-${task.priority}">${task.priority}</span>`;
  if (task.recurring && task.recurring !== 'none') {
    badges += `<span class="badge badge-recurring">${RECURRING_ICONS[task.recurring]} ${RECURRING_LABELS[task.recurring]}</span>`;
  }
  if (task.timer_duration_minutes && !hasTimer) {
    badges += `<span class="badge badge-timer">⏱ ${formatDuration(task.timer_duration_minutes)}</span>`;
  }

  // Timer controls
  let timerControls = '';
  if (hasTimer && remainingMs > 0) {
    timerControls = `
      <div class="task-timer-countdown">
        <span class="timer-dot"></span>
        <span data-timer-id="${escHtml(task.id)}">${formatCountdown(remainingMs)}</span>
      </div>
      <button class="timer-start-btn timer-stop-btn" onclick="stopTimer('${escHtml(task.id)}')">⏹ Stop</button>
    `;
  } else if (task.timer_duration_minutes && !task.completed) {
    timerControls = `<button class="timer-start-btn" onclick='startTimer(${JSON.stringify(task)})'>▶ Start ${formatDuration(task.timer_duration_minutes)} timer</button>`;
  }

  card.innerHTML = `
    <div class="task-checkbox-wrap">
      <input type="checkbox" class="task-checkbox" data-check="${escHtml(task.id)}"
        ${task.completed ? 'checked' : ''}
        aria-label="Mark ${escHtml(task.title)} ${task.completed ? 'incomplete' : 'complete'}"
        onchange="toggleComplete('${escHtml(task.id)}', ${task.completed})">
    </div>
    <div class="task-body">
      <div class="task-title-row">
        <span class="task-title">${escHtml(task.title)}</span>
        ${badges}
      </div>
      ${task.description ? `<p class="task-description">${escHtml(task.description)}</p>` : ''}
      <div class="task-meta">
        <span class="task-date">📅 ${created}</span>
        ${dueInfo ? `<span class="task-date${overdue ? ' overdue' : ''}">⏰ Due ${dueInfo.label}</span>` : ''}
        ${timerControls}
      </div>
    </div>
    <div class="task-actions">
      <button class="action-btn" onclick="openEditModal('${escHtml(task.id)}')" title="Edit">✏️</button>
      <button class="action-btn delete" onclick="deleteTask('${escHtml(task.id)}')" title="Delete">🗑️</button>
    </div>
  `;
  return card;
}

function updateStats() {
  const total  = allTasks.length;
  const done   = allTasks.filter(t => t.completed).length;
  const active = total - done;
  const high   = allTasks.filter(t => t.priority === 'high' && !t.completed).length;
  animateNumber('stat-total-num',  total);
  animateNumber('stat-active-num', active);
  animateNumber('stat-done-num',   done);
  animateNumber('stat-high-num',   high);
}

function animateNumber(elId, target) {
  const el = document.getElementById(elId);
  if (!el) return;
  const current = parseInt(el.textContent, 10) || 0;
  if (current === target) return;
  const step = target > current ? 1 : -1;
  const steps = Math.abs(target - current);
  const delay = Math.min(30, 300 / steps);
  let count = current;
  const timer = setInterval(() => {
    count += step;
    el.textContent = count;
    if (count === target) clearInterval(timer);
  }, delay);
}

// ═══════════════════════════════════════════════════════════════════════════
// Edit Modal
// ═══════════════════════════════════════════════════════════════════════════

function openEditModal(taskId) {
  const task = allTasks.find(t => t.id === taskId);
  if (!task) return;

  document.getElementById('edit-task-id').value   = task.id;
  document.getElementById('edit-title').value     = task.title;
  document.getElementById('edit-desc').value      = task.description || '';
  document.getElementById('edit-priority').value  = task.priority;
  document.getElementById('edit-recurring').value = task.recurring || 'none';
  document.getElementById('edit-timer').value     = task.timer_duration_minutes || '';
  document.getElementById('edit-due').value       = task.due_date
    ? new Date(task.due_date).toISOString().split('T')[0] : '';
  showError(document.getElementById('edit-error'), '');

  const modal = document.getElementById('edit-modal');
  modal.classList.remove('hidden');
  document.getElementById('edit-title').focus();
  modal.onclick = e => { if (e.target === modal) closeEditModal(); };
}

function closeEditModal() {
  document.getElementById('edit-modal').classList.add('hidden');
}

document.addEventListener('keydown', e => {
  if (e.key === 'Escape') { closeEditModal(); if (addFormOpen) toggleAddForm(); }
});

// ═══════════════════════════════════════════════════════════════════════════
// Screen transitions
// ═══════════════════════════════════════════════════════════════════════════

function showAuth() {
  document.getElementById('app-screen').classList.add('hidden');
  document.getElementById('auth-screen').classList.remove('hidden');
}

function showApp() {
  document.getElementById('auth-screen').classList.add('hidden');
  document.getElementById('app-screen').classList.remove('hidden');
  document.getElementById('user-email').textContent = currentUser?.email || '';
  requestNotificationPermission();
  loadTasks();
}

function switchTab(tab) {
  const isLogin = tab === 'login';
  document.getElementById('login-form').classList.toggle('hidden', !isLogin);
  document.getElementById('register-form').classList.toggle('hidden', isLogin);
  document.getElementById('tab-login').classList.toggle('active', isLogin);
  document.getElementById('tab-register').classList.toggle('active', !isLogin);
}

// ═══════════════════════════════════════════════════════════════════════════
// UI Helpers
// ═══════════════════════════════════════════════════════════════════════════

function setLoading(btn, loading) {
  btn.disabled = loading;
  btn.querySelector('.btn-text').classList.toggle('hidden', loading);
  btn.querySelector('.btn-spinner').classList.toggle('hidden', !loading);
}

function showError(el, message) {
  if (!el) return;
  if (message) { el.textContent = message; el.classList.remove('hidden'); }
  else { el.classList.add('hidden'); }
}

function showLoading(show) {
  document.getElementById('loading-state').classList.toggle('hidden', !show);
  document.getElementById('task-list').classList.toggle('hidden', show);
}

let toastTimer;
function toast(message, type = 'success') {
  const el = document.getElementById('toast');
  el.textContent = message;
  el.className = `toast ${type}`;
  el.classList.remove('hidden');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => el.classList.add('hidden'), 3500);
}

function formatDueDate(isoString) {
  const date  = new Date(isoString);
  const today = new Date(); today.setHours(0,0,0,0);
  const d     = new Date(date); d.setHours(0,0,0,0);
  const diff  = Math.floor((d - today) / 86400000);
  if (diff < 0)   return { label:`${Math.abs(diff)}d overdue`, overdue:true };
  if (diff === 0) return { label:'Today',    overdue:false };
  if (diff === 1) return { label:'Tomorrow', overdue:false };
  return { label:date.toLocaleDateString('en-US',{month:'short',day:'numeric'}), overdue:false };
}

function escHtml(str) {
  return String(str)
    .replace(/&/g,'&amp;').replace(/</g,'&lt;')
    .replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

// ═══════════════════════════════════════════════════════════════════════════
// Bootstrap
// ═══════════════════════════════════════════════════════════════════════════

(function init() {
  if (token && currentUser) { showApp(); }
  else { showAuth(); }
})();
