<script>
  import { onMount, onDestroy } from 'svelte';
  import { GetOpenIncidents, GetResolvedIncidents, GetServicesConfig, SetSelectedServices } from '../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff, BrowserOpenURL, LogDebug } from '../../wailsjs/runtime/runtime.js';

  let activeTab = 'open';
  let incidents = [];
  let filteredIncidents = [];
  let servicesConfig = null;
  let selectedServices = new Set();
  let showServiceDropdown = false;
  let loading = false;
  let dropdownRef;

  onMount(async () => {
    LogDebug("IncidentsPanel mounted");
    
    // Load services configuration
    try {
      servicesConfig = await GetServicesConfig();
      LogDebug(`Services config loaded: ${JSON.stringify(servicesConfig)}`);
      
      // Initialize all services as selected
      if (servicesConfig && servicesConfig.services) {
        servicesConfig.services.forEach(service => {
          selectedServices.add(service.name);
        });
        selectedServices = selectedServices;
        updateSelectedServicesBackend();
      }
    } catch (err) {
      LogDebug(`No services configuration found: ${err}`);
      console.log('No services configuration found');
    }

    // Load initial incidents
    loadIncidents();

    // Listen for incident updates
    EventsOn('incidents-updated', (type) => {
      LogDebug(`Incidents update event received: ${type}`);
      if (type === 'open' && activeTab === 'open') {
        loadIncidents();
      }
    });

    // Add click outside handler
    document.addEventListener('click', handleClickOutside);
    document.addEventListener('mousedown', logMouseEvent);
    document.addEventListener('mouseup', logMouseEvent);
  });

  onDestroy(() => {
    EventsOff('incidents-updated');
    document.removeEventListener('click', handleClickOutside);
    document.removeEventListener('mousedown', logMouseEvent);
    document.removeEventListener('mouseup', logMouseEvent);
  });

  function logMouseEvent(event) {
    const target = event.target;
    const classList = target.classList ? Array.from(target.classList).join(', ') : 'no-classes';
    LogDebug(`Mouse ${event.type} on element: ${target.tagName}, classes: ${classList}, id: ${target.id || 'no-id'}`);
    console.log(`Mouse ${event.type}:`, target);
  }

  function handleClickOutside(event) {
    if (dropdownRef && !dropdownRef.contains(event.target)) {
      showServiceDropdown = false;
    }
  }

  async function loadIncidents() {
    LogDebug(`Loading incidents for tab: ${activeTab}`);
    loading = true;
    try {
      if (activeTab === 'open') {
        incidents = await GetOpenIncidents();
        LogDebug(`Loaded ${incidents.length} open incidents`);
      } else {
        incidents = await GetResolvedIncidents();
        LogDebug(`Loaded ${incidents.length} resolved incidents`);
      }
      filterIncidents();
    } catch (err) {
      LogDebug(`Failed to load incidents: ${err}`);
      console.error('Failed to load incidents:', err);
      incidents = [];
      filteredIncidents = [];
    }
    loading = false;
  }

  function filterIncidents() {
    if (!servicesConfig || selectedServices.size === 0) {
      filteredIncidents = incidents;
      LogDebug(`No filter applied, showing all ${incidents.length} incidents`);
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
    LogDebug(`Filtered to ${filteredIncidents.length} incidents from ${incidents.length} total`);
  }

  function toggleService(serviceName) {
    LogDebug(`Toggling service: ${serviceName}`);
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
    
    LogDebug(`Updating selected services in backend: ${serviceIds.join(', ')}`);
    await SetSelectedServices(serviceIds);
  }

  function switchTab(tab) {
    LogDebug(`Switching to tab: ${tab}`);
    console.log('Tab switch clicked:', tab);
    activeTab = tab;
    loadIncidents();
  }

  function toggleDropdown(event) {
    LogDebug(`Dropdown toggle clicked, current state: ${showServiceDropdown}`);
    console.log('Dropdown toggle clicked');
    event.stopPropagation();
    event.preventDefault();
    showServiceDropdown = !showServiceDropdown;
    LogDebug(`Dropdown new state: ${showServiceDropdown}`);
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

  async function openIncident(url) {
    LogDebug(`Opening incident URL: ${url}`);
    console.log('Opening URL:', url);
    try {
      await BrowserOpenURL(url);
      LogDebug(`Successfully opened URL`);
    } catch (err) {
      LogDebug(`Failed to open URL: ${err}`);
      console.error('Failed to open incident URL:', err);
    }
  }

  function handleTabClick(tab) {
    LogDebug(`Tab clicked: ${tab}`);
    console.log('Tab clicked:', tab);
    switchTab(tab);
  }

  function handleIncidentClick(incident) {
    LogDebug(`Incident clicked: #${incident.IncidentNumber}`);
    console.log('Incident clicked:', incident);
    openIncident(incident.HTMLURL);
  }
</script>

<div class="incidents-panel" style="--wails-draggable: no-drag;">
  <div class="panel-header" style="--wails-draggable: no-drag;">
    <h2>Incidents</h2>
    
    <div class="controls" style="--wails-draggable: no-drag;">
      <div class="service-selector" bind:this={dropdownRef} style="--wails-draggable: no-drag;">
        <button 
          class="service-dropdown-btn" 
          on:click={toggleDropdown}
          on:mousedown|preventDefault|stopPropagation
          on:mouseup|preventDefault|stopPropagation
          type="button"
          style="--wails-draggable: no-drag;">
          Services ({selectedServices.size})
          <span class="dropdown-arrow">▼</span>
        </button>
        
        {#if showServiceDropdown}
          <div class="service-dropdown" style="--wails-draggable: no-drag;">
            {#if servicesConfig && servicesConfig.services}
              {#each servicesConfig.services as service}
                <label class="service-option" style="--wails-draggable: no-drag;">
                  <input 
                    type="checkbox" 
                    checked={selectedServices.has(service.name)}
                    on:change={() => toggleService(service.name)}
                    style="--wails-draggable: no-drag;"
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
      
      <div class="tabs" style="--wails-draggable: no-drag;">
        <button 
          class="tab {activeTab === 'open' ? 'active' : ''}" 
          on:click={() => handleTabClick('open')}
          on:mousedown|preventDefault|stopPropagation
          on:mouseup|preventDefault|stopPropagation
          type="button"
          style="--wails-draggable: no-drag;">
          Open Incidents
        </button>
        <button 
          class="tab {activeTab === 'resolved' ? 'active' : ''}" 
          on:click={() => handleTabClick('resolved')}
          on:mousedown|preventDefault|stopPropagation
          on:mouseup|preventDefault|stopPropagation
          type="button"
          style="--wails-draggable: no-drag;">
          Resolved
        </button>
      </div>
    </div>
  </div>

  <div class="incidents-list" style="--wails-draggable: no-drag;">
    {#if loading}
      <div class="loading">Loading incidents...</div>
    {:else if filteredIncidents.length === 0}
      <div class="no-incidents">No incidents found</div>
    {:else}
      {#each filteredIncidents as incident}
        <div 
          class="incident-card" 
          on:click={() => handleIncidentClick(incident)}
          on:mousedown|preventDefault|stopPropagation
          on:mouseup|preventDefault|stopPropagation
          on:keydown={(e) => e.key === 'Enter' && handleIncidentClick(incident)}
          role="button"
          tabindex="0"
          style="--wails-draggable: no-drag;">
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
        </div>
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
    font-size: 14px;
    user-select: none;
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
    z-index: 1000;
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.3);
  }

  .service-option {
    display: flex;
    align-items: center;
    padding: 10px;
    cursor: pointer;
    transition: background 0.2s;
    user-select: none;
  }

  .service-option:hover {
    background: #333;
  }

  .service-option input {
    margin-right: 10px;
    cursor: pointer;
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
    font-size: 14px;
    user-select: none;
  }

  .tab:hover {
    background: #444;
    color: #fff;
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
    font-size: inherit;
    user-select: none;
  }

  .incident-card:hover {
    background: #333;
    border-color: #444;
    transform: translateX(2px);
  }

  .incident-card:focus {
    outline: 2px solid #007bff;
    outline-offset: 2px;
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