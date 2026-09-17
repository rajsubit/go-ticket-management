// API Client for Ticket Management REST API

const API_BASE = '/api/v1';

/**
 * Generic request helper handling JSON serialization and error parsing
 */
async function request(endpoint, options = {}) {
  const url = `${API_BASE}${endpoint}`;
  const defaultHeaders = {
    'Content-Type': 'application/json',
  };

  const config = {
    ...options,
    headers: {
      ...defaultHeaders,
      ...options.headers,
    },
  };

  try {
    const response = await fetch(url, config);
    const result = await response.json();

    if (!response.ok || !result.success) {
      const errorMsg = result.error || `HTTP error ${response.status}: ${response.statusText}`;
      throw new Error(errorMsg);
    }

    return result.data;
  } catch (err) {
    console.error(`API Error on ${config.method || 'GET'} ${url}:`, err);
    throw err;
  }
}

export const API = {
  /**
   * Fetch all tickets with optional status & priority filters
   */
  async getTickets(filters = {}) {
    const params = new URLSearchParams();
    if (filters.status) params.append('status', filters.status);
    if (filters.priority) params.append('priority', filters.priority);

    const queryString = params.toString() ? `?${params.toString()}` : '';
    return request(`/tickets${queryString}`);
  },

  /**
   * Fetch aggregated ticket metrics
   */
  async getMetrics() {
    return request('/tickets/metrics');
  },

  /**
   * Get single ticket by ID
   */
  async getTicket(id) {
    return request(`/tickets/${encodeURIComponent(id)}`);
  },

  /**
   * Create a new ticket
   */
  async createTicket(ticketData) {
    return request('/tickets', {
      method: 'POST',
      body: JSON.stringify(ticketData),
    });
  },

  /**
   * Update fields of an existing ticket
   */
  async updateTicket(id, updates) {
    return request(`/tickets/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  },

  /**
   * Transition ticket status
   */
  async updateStatus(id, status) {
    return request(`/tickets/${encodeURIComponent(id)}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    });
  },

  /**
   * Delete a ticket by ID
   */
  async deleteTicket(id) {
    return request(`/tickets/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
  },
};
