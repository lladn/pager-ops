<script>
  import { onMount, onDestroy } from 'svelte';
  import { GetOpenIncidents, GetResolvedIncidents, GetServicesConfig, SetSelectedServices } from '../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';

  let activeTab = 'open';
  let incidents = [];
  let filteredIncidents = [];
  let servicesConfig = null;
  let selectedServices = new Set();
  let showServiceDropdown = false;
  let loading = false;

  onMount(async () => {
    // Load services configuration
    try {
      servicesConfig = await GetServicesConfig();
      // Initialize all services as selected
      if (servicesConfig && servicesConfig.services) {
        servicesConfig.services.forEach(service => {
          selectedServices.add(service.name);
        });
        selectedServices = selectedServices;
        updateSelectedServicesBackend();
      }
    } catch (err) {
      console.log('No services configuration found');
    }

    // Load initial incidents
    loadIncidents();

    // Listen for incident updates
    EventsOn('incidents-updated', (type) => {
      if (type === 'open' && activeTab === 'open') {
        loadIncidents();
      }
    });
  });

  onDestroy(() => {
    EventsOff('incidents-updated');
  });

  async function loadIncidents() {
    loading = true;
    try {
      if (activeTab === 'open') {
        incidents = await GetOpenIncidents();
      } else {
        incidents = await GetResolvedIncidents();
      }
      filterIncidents();
    } catch (err) {
      console.error('Failed to load incidents:', err);
      incidents = [];
    }
    loading = false;
  }

  function filterIncidents() {
    if (!servicesConfig || selectedServices.size === 0) {
      filteredIncidents = incidents;
      return;
    }

    // Get all service IDs for selected service names
    const selectedServiceIds = new Set();
    servicesConfig.services.forEach(service => {
      if (selectedServices.has(service.name)) {
        if (typeof service.id === 'string') {
          selectedServiceIds.add(service.id);
        } else if (Array.isArray(service.id)) {
          service.id.forEach(id => selectedServiceIds.add(id));
        }
      }
    });

    filteredIncidents = incidents.filter(incident => 
      selectedServiceIds.has(incident.ServiceID)
    );
  }

  function toggleService(serviceName) {
    if (selectedServices.has(serviceName)) {
      selectedServices.delete(serviceName);
    } else {
      selectedServices.add(serviceName);
    }
    selectedServices = selectedServices;
    updateSelectedServicesBackend();
    filterIncidents();
  }

  async function updateSelectedServicesBackend() {
    if (!servicesConfig) return;
    
    const serviceIds = [];
    servicesConfig.services.forEach(service => {
      if (selectedServices.has(service.name)) {
        if (typeof service.id === 'string') {
          serviceIds.push(service.id);
        } else if (Array.isArray(service.id)) {
          serviceIds.push(...service.id);
        }
      }
    });
    
    await SetSelectedServices(serviceIds);
  }

  function switchTab(tab) {
    activeTab = tab;
    loadIncidents();
  }

  function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString();
  }

  function getStatusColor(status) {
    switch(status) {
      case 'triggered': return '#dc3545';
      case 'acknowledged': return '#ffc107';
      case 'resolved': return '#28a745';
      default: return '#6c757d';
    }
  }

  function openIncident(url) {
    window.open(url, '_blank');
  }
</script>

<div class="incidents-panel">
  <div class="panel-header">
    <h2>Incidents</h2>
    
    <div class="controls">
      <div class="service-selector">
        <button 
          class="service-dropdown-btn" 
          on:click={() => showServiceDropdown = !showServiceDropdown}>
          Services ({selectedServices.size})
          <span class="dropdown-arrow">▼</span>
        </button>
        
        {#if showServiceDropdown}
          <div class="service-dropdown">
            {#if servicesConfig && servicesConfig.services}
              {#each servicesConfig.services as service}
                <label class="service-option">
                  <input 
                    type="checkbox" 
                    checked={selectedServices.has(service.name)}
                    on:change={() => toggleService(service.name)}
                  />
                  <span>{service.name}</span>
                </label>
              {/each}
            {:else}
              <p class="no-services">No services configured</p>
            {/if}
          </div>
        {/if}
      </div>
      
      <div class="tabs">
        <button 
          class="tab {activeTab === 'open' ? 'active' : ''}" 
          on:click={() => switchTab('open')}>
          Open Incidents
        </button>
        <button 
          class="tab {activeTab === 'resolved' ? 'active' : ''}" 
          on:click={() => switchTab('resolved')}>
          Resolved
        </button>
      </div>
    </div>
  </div>

  <div class="incidents-list">
    {#if loading}
      <div class="loading">Loading incidents...</div>
    {:else if filteredIncidents.length === 0}
      <div class="no-incidents">No incidents found</div>
    {:else}
      {#each filteredIncidents as incident}
        <button class="incident-card" on:click={() => openIncident(incident.HTMLURL)}>
          <div class="incident-header">
            <span class="incident-number">#{incident.IncidentNumber}</span>
            <span 
              class="incident-status" 
              style="background-color: {getStatusColor(incident.Status)}">
              {incident.Status}
            </span>
          </div>
          
          <h3 class="incident-title">{incident.Title}</h3>
          
          <div class="incident-details">
            <div class="detail-item">
              <span class="label">Service:</span>
              <span class="value">{incident.ServiceSummary}</span>
            </div>
            <div class="detail-item">
              <span class="label">Created:</span>
              <span class="value">{formatDate(incident.CreatedAt)}</span>
            </div>
            {#if incident.AlertCount > 0}
              <div class="detail-item">
                <span class="label">Alerts:</span>
                <span class="value">{incident.AlertCount}</span>
              </div>
            {/if}
          </div>
        </button>
      {/each}
    {/if}
  </div>
</div>

<style>
  .incidents-panel {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #1a1a1a;
    color: #fff;
  }

  .panel-header {
    padding: 20px;
    border-bottom: 1px solid #333;
    background: #222;
  }

  .panel-header h2 {
    margin: 0 0 15px 0;
  }

  .controls {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .service-selector {
    position: relative;
  }

  .service-dropdown-btn {
    background: #333;
    color: #fff;
    border: 1px solid #444;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .service-dropdown-btn:hover {
    background: #444;
  }

  .dropdown-arrow {
    font-size: 10px;
  }

  .service-dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    margin-top: 4px;
    background: #2a2a2a;
    border: 1px solid #444;
    border-radius: 4px;
    min-width: 200px;
    max-height: 300px;
    overflow-y: auto;
    z-index: 100;
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.3);
  }

  .service-option {
    display: flex;
    align-items: center;
    padding: 10px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .service-option:hover {
    background: #333;
  }

  .service-option input {
    margin-right: 10px;
  }

  .no-services {
    padding: 10px;
    color: #999;
    text-align: center;
  }

  .tabs {
    display: flex;
    gap: 10px;
  }

  .tab {
    background: #333;
    color: #999;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.3s;
  }

  .tab:hover {
    background: #444;
  }

  .tab.active {
    background: #007bff;
    color: #fff;
  }

  .incidents-list {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
  }

  .loading, .no-incidents {
    text-align: center;
    color: #999;
    padding: 40px;
  }

  .incident-card {
    display: block;
    width: 100%;
    text-align: left;
    background: #2a2a2a;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 16px;
    margin-bottom: 12px;
    cursor: pointer;
    transition: all 0.3s;
    color: inherit;
    font-family: inherit;
  }

  .incident-card:hover {
    background: #333;
    border-color: #444;
    transform: translateX(2px);
  }

  .incident-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;
  }

  .incident-number {
    color: #999;
    font-size: 14px;
  }

  .incident-status {
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 12px;
    text-transform: uppercase;
    font-weight: bold;
    color: #fff;
  }

  .incident-title {
    margin: 0 0 12px 0;
    font-size: 16px;
    line-height: 1.4;
  }

  .incident-details {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .detail-item {
    display: flex;
    font-size: 14px;
  }

  .label {
    color: #999;
    margin-right: 8px;
    min-width: 70px;
  }

  .value {
    color: #ccc;
  }
</style>