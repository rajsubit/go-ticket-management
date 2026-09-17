import { API } from './api.js';

// Application State
const state = {
  tickets: [],
  metrics: { total: 0, open: 0, in_progress: 0, resolved: 0, closed: 0 },
  currentView: 'kanban', // 'kanban' | 'list'
  searchQuery: '',
  statusFilter: '',
  priorityFilter: '',
  editingTicketId: null,
};

// Legal state machine transitions (matching Go domain rules)
const LEGAL_TRANSITIONS = {
  OPEN: ['IN_PROGRESS', 'CLOSED'],
  IN_PROGRESS: ['RESOLVED', 'OPEN', 'CLOSED'],
  RESOLVED: ['CLOSED', 'IN_PROGRESS'],
  CLOSED: ['OPEN'],
};

// DOM Element Selectors
const elements = {
  totalCount: document.getElementById('metricTotal'),
  openCount: document.getElementById('metricOpen'),
  inProgressCount: document.getElementById('metricInProgress'),
  resolvedCount: document.getElementById('metricResolved'),
  closedCount: document.getElementById('metricClosed'),
  kanbanView: document.getElementById('kanbanView'),
  listView: document.getElementById('listView'),
  tableBody: document.getElementById('ticketTableBody'),
  searchInput: document.getElementById('searchInput'),
  statusFilter: document.getElementById('statusFilter'),
  priorityFilter: document.getElementById('priorityFilter'),
  btnViewKanban: document.getElementById('btnViewKanban'),
  btnViewList: document.getElementById('btnViewList'),
  btnNewTicket: document.getElementById('btnNewTicket'),
  btnRefresh: document.getElementById('btnRefresh'),
  ticketModal: document.getElementById('ticketModal'),
  ticketForm: document.getElementById('ticketForm'),
  modalTitle: document.getElementById('modalTitle'),
  btnCloseModal: document.getElementById('btnCloseModal'),
  btnCancelModal: document.getElementById('btnCancelModal'),
  toastContainer: document.getElementById('toastContainer'),
};

// ==========================================================================
// TOAST NOTIFICATIONS
// ==========================================================================
function showToast(message, type = 'info') {
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;

  const icon = type === 'success' ? '✓' : type === 'error' ? '✕' : 'ℹ';
  toast.innerHTML = `
    <span style="font-weight:700;">${icon}</span>
    <span class="toast-message">${escapeHtml(message)}</span>
  `;

  elements.toastContainer.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transform = 'translateY(10px)';
    setTimeout(() => toast.remove(), 250);
  }, 3500);
}

// ==========================================================================
// DATA LOADING & REFRESH
// ==========================================================================
async function loadData() {
  try {
    const [tickets, metrics] = await Promise.all([
      API.getTickets({
        status: state.statusFilter,
        priority: state.priorityFilter,
      }),
      API.getMetrics(),
    ]);

    state.tickets = tickets || [];
    state.metrics = metrics || { total: 0, open: 0, in_progress: 0, resolved: 0, closed: 0 };

    renderMetrics();
    renderView();
  } catch (err) {
    showToast(`Failed to fetch data: ${err.message}`, 'error');
  }
}

function renderMetrics() {
  elements.totalCount.textContent = state.metrics.total;
  elements.openCount.textContent = state.metrics.open;
  elements.inProgressCount.textContent = state.metrics.in_progress;
  elements.resolvedCount.textContent = state.metrics.resolved;
  elements.closedCount.textContent = state.metrics.closed;
}

function getFilteredTickets() {
  let list = state.tickets;

  if (state.searchQuery) {
    const q = state.searchQuery.toLowerCase();
    list = list.filter(
      (t) =>
        t.title.toLowerCase().includes(q) ||
        (t.description && t.description.toLowerCase().includes(q)) ||
        (t.assignee && t.assignee.toLowerCase().includes(q)) ||
        t.id.toLowerCase().includes(q)
    );
  }

  return list;
}

function renderView() {
  const filtered = getFilteredTickets();
  if (state.currentView === 'kanban') {
    elements.kanbanView.style.display = 'grid';
    elements.listView.style.display = 'none';
    renderKanban(filtered);
  } else {
    elements.kanbanView.style.display = 'none';
    elements.listView.style.display = 'block';
    renderList(filtered);
  }
}

// ==========================================================================
// KANBAN BOARD RENDERING & DRAG-AND-DROP
// ==========================================================================
function renderKanban(tickets) {
  const lanes = {
    OPEN: document.getElementById('laneOpen'),
    IN_PROGRESS: document.getElementById('laneInProgress'),
    RESOLVED: document.getElementById('laneResolved'),
    CLOSED: document.getElementById('laneClosed'),
  };

  const badgeCounts = {
    OPEN: document.getElementById('badgeOpen'),
    IN_PROGRESS: document.getElementById('badgeInProgress'),
    RESOLVED: document.getElementById('badgeResolved'),
    CLOSED: document.getElementById('badgeClosed'),
  };

  // Clear lane contents
  Object.keys(lanes).forEach((status) => {
    lanes[status].innerHTML = '';
  });

  const counts = { OPEN: 0, IN_PROGRESS: 0, RESOLVED: 0, CLOSED: 0 };

  tickets.forEach((ticket) => {
    const lane = lanes[ticket.status];
    if (lane) {
      counts[ticket.status]++;
      lane.appendChild(createTicketCard(ticket));
    }
  });

  // Update badge counts and empty states
  Object.keys(lanes).forEach((status) => {
    badgeCounts[status].textContent = counts[status];
    if (counts[status] === 0) {
      lanes[status].innerHTML = `
        <div class="empty-column">
          <span>No tickets</span>
        </div>
      `;
    }
  });
}

function createTicketCard(ticket) {
  const card = document.createElement('div');
  card.className = 'ticket-card';
  card.draggable = true;
  card.id = `card-${ticket.id}`;
  card.dataset.id = ticket.id;
  card.dataset.status = ticket.status;

  const initials = ticket.assignee ? ticket.assignee.substring(0, 2).toUpperCase() : '??';
  const allowed = LEGAL_TRANSITIONS[ticket.status] || [];

  const transitionsHtml = allowed
    .map(
      (target) => `
      <button class="transition-btn" data-id="${ticket.id}" data-target="${target}">
        → ${formatStatusName(target)}
      </button>`
    )
    .join('');

  card.innerHTML = `
    <div class="card-top">
      <span class="badge-priority ${ticket.priority}">${ticket.priority}</span>
      <span class="ticket-id">${ticket.id}</span>
    </div>
    <div class="card-title">${escapeHtml(ticket.title)}</div>
    ${ticket.description ? `<div class="card-desc">${escapeHtml(ticket.description)}</div>` : ''}
    <div class="card-footer">
      <div class="assignee-wrap">
        <div class="avatar">${initials}</div>
        <span>${escapeHtml(ticket.assignee || 'Unassigned')}</span>
      </div>
      <div class="card-actions">
        <button class="action-btn edit" data-id="${ticket.id}" title="Edit Ticket">✎</button>
        <button class="action-btn delete" data-id="${ticket.id}" title="Delete Ticket">🗑</button>
      </div>
    </div>
    ${transitionsHtml ? `<div class="transition-menu">${transitionsHtml}</div>` : ''}
  `;

  // Attach Drag Events
  card.addEventListener('dragstart', handleDragStart);
  card.addEventListener('dragend', handleDragEnd);

  return card;
}

// Drag Handlers
let draggedCardId = null;

function handleDragStart(e) {
  draggedCardId = this.dataset.id;
  this.classList.add('dragging');
  e.dataTransfer.setData('text/plain', draggedCardId);
  e.dataTransfer.effectAllowed = 'move';
}

function handleDragEnd() {
  this.classList.remove('dragging');
  draggedCardId = null;
  document.querySelectorAll('.kanban-column').forEach((col) => {
    col.classList.remove('drag-over');
  });
}

function setupKanbanLanes() {
  document.querySelectorAll('.kanban-column').forEach((column) => {
    column.addEventListener('dragover', (e) => {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      column.classList.add('drag-over');
    });

    column.addEventListener('dragleave', () => {
      column.classList.remove('drag-over');
    });

    column.addEventListener('drop', async (e) => {
      e.preventDefault();
      column.classList.remove('drag-over');
      const targetStatus = column.dataset.status;
      const ticketId = e.dataTransfer.getData('text/plain');

      if (!ticketId || !targetStatus) return;

      const currentTicket = state.tickets.find((t) => t.id === ticketId);
      if (currentTicket && currentTicket.status === targetStatus) return;

      await handleStatusTransition(ticketId, targetStatus);
    });
  });
}

async function handleStatusTransition(ticketId, targetStatus) {
  try {
    await API.updateStatus(ticketId, targetStatus);
    showToast(`Status updated to ${formatStatusName(targetStatus)}`, 'success');
    await loadData();
  } catch (err) {
    showToast(err.message, 'error');
  }
}

// ==========================================================================
// LIST VIEW RENDERING
// ==========================================================================
function renderList(tickets) {
  if (tickets.length === 0) {
    elements.tableBody.innerHTML = `
      <tr>
        <td colspan="7" style="text-align: center; padding: 3rem; color: var(--text-muted);">
          No tickets matching current filters.
        </td>
      </tr>
    `;
    return;
  }

  elements.tableBody.innerHTML = tickets
    .map(
      (ticket) => `
      <tr>
        <td style="font-family: monospace; font-size: 0.8rem;">${ticket.id}</td>
        <td style="font-weight: 600; color: var(--text-primary);">${escapeHtml(ticket.title)}</td>
        <td><span class="badge-status ${ticket.status}">${formatStatusName(ticket.status)}</span></td>
        <td><span class="badge-priority ${ticket.priority}">${ticket.priority}</span></td>
        <td>${escapeHtml(ticket.assignee || '—')}</td>
        <td style="font-size: 0.775rem; color: var(--text-muted);">${formatDate(ticket.created_at)}</td>
        <td>
          <div class="card-actions">
            <button class="action-btn edit" data-id="${ticket.id}" title="Edit">✎</button>
            <button class="action-btn delete" data-id="${ticket.id}" title="Delete">🗑</button>
          </div>
        </td>
      </tr>
    `
    )
    .join('');
}

// ==========================================================================
// MODAL & FORM CONTROLS
// ==========================================================================
function openCreateModal() {
  state.editingTicketId = null;
  elements.modalTitle.textContent = 'Create New Ticket';
  elements.ticketForm.reset();
  elements.ticketModal.classList.add('active');
  document.getElementById('ticketTitle').focus();
}

function openEditModal(id) {
  const t = state.tickets.find((item) => item.id === id);
  if (!t) return;

  state.editingTicketId = id;
  elements.modalTitle.textContent = `Edit Ticket (${t.id})`;
  document.getElementById('ticketTitle').value = t.title;
  document.getElementById('ticketDesc').value = t.description || '';
  document.getElementById('ticketPriority').value = t.priority;
  document.getElementById('ticketAssignee').value = t.assignee || '';

  elements.ticketModal.classList.add('active');
}

function closeModal() {
  elements.ticketModal.classList.remove('active');
  state.editingTicketId = null;
  elements.ticketForm.reset();
}

async function handleFormSubmit(e) {
  e.preventDefault();

  const title = document.getElementById('ticketTitle').value.trim();
  const description = document.getElementById('ticketDesc').value.trim();
  const priority = document.getElementById('ticketPriority').value;
  const assignee = document.getElementById('ticketAssignee').value.trim();

  if (title.length < 3 || title.length > 100) {
    showToast('Title must be between 3 and 100 characters', 'error');
    return;
  }

  try {
    if (state.editingTicketId) {
      await API.updateTicket(state.editingTicketId, {
        title: title,
        description: description,
        priority: priority,
        assignee: assignee,
      });
      showToast('Ticket updated successfully', 'success');
    } else {
      await API.createTicket({
        title,
        description,
        priority,
        assignee,
      });
      showToast('Ticket created successfully', 'success');
    }

    closeModal();
    await loadData();
  } catch (err) {
    showToast(err.message, 'error');
  }
}

async function handleDelete(id) {
  if (!confirm(`Are you sure you want to delete ticket ${id}?`)) {
    return;
  }

  try {
    await API.deleteTicket(id);
    showToast('Ticket deleted', 'info');
    await loadData();
  } catch (err) {
    showToast(`Failed to delete: ${err.message}`, 'error');
  }
}

// ==========================================================================
// EVENT LISTENERS & INITIALIZATION
// ==========================================================================
function setupEventListeners() {
  // View Switchers
  elements.btnViewKanban.addEventListener('click', () => {
    state.currentView = 'kanban';
    elements.btnViewKanban.classList.add('active');
    elements.btnViewList.classList.remove('active');
    renderView();
  });

  elements.btnViewList.addEventListener('click', () => {
    state.currentView = 'list';
    elements.btnViewList.classList.add('active');
    elements.btnViewKanban.classList.remove('active');
    renderView();
  });

  // Search & Filters
  elements.searchInput.addEventListener('input', (e) => {
    state.searchQuery = e.target.value;
    renderView();
  });

  elements.statusFilter.addEventListener('change', (e) => {
    state.statusFilter = e.target.value;
    loadData();
  });

  elements.priorityFilter.addEventListener('change', (e) => {
    state.priorityFilter = e.target.value;
    loadData();
  });

  // Action Buttons
  elements.btnNewTicket.addEventListener('click', openCreateModal);
  elements.btnRefresh.addEventListener('click', () => {
    loadData();
    showToast('Data refreshed', 'info');
  });

  elements.btnCloseModal.addEventListener('click', closeModal);
  elements.btnCancelModal.addEventListener('click', closeModal);
  elements.ticketForm.addEventListener('submit', handleFormSubmit);

  // Close modal when clicking on backdrop
  elements.ticketModal.addEventListener('click', (e) => {
    if (e.target === elements.ticketModal) closeModal();
  });

  // Delegation for edit, delete, and transition buttons
  document.addEventListener('click', (e) => {
    const editBtn = e.target.closest('.action-btn.edit');
    if (editBtn) {
      openEditModal(editBtn.dataset.id);
      return;
    }

    const delBtn = e.target.closest('.action-btn.delete');
    if (delBtn) {
      handleDelete(delBtn.dataset.id);
      return;
    }

    const transBtn = e.target.closest('.transition-btn');
    if (transBtn) {
      handleStatusTransition(transBtn.dataset.id, transBtn.dataset.target);
      return;
    }
  });

  setupKanbanLanes();
}

// Utility Helpers
function formatStatusName(status) {
  switch (status) {
    case 'OPEN': return 'Open';
    case 'IN_PROGRESS': return 'In Progress';
    case 'RESOLVED': return 'Resolved';
    case 'CLOSED': return 'Closed';
    default: return status;
  }
}

function formatDate(isoString) {
  if (!isoString) return '—';
  const d = new Date(isoString);
  return d.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function escapeHtml(str) {
  if (!str) return '';
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

// Initial Boot
document.addEventListener('DOMContentLoaded', () => {
  setupEventListeners();
  loadData();
});
