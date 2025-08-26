<script>
  import { onMount, onDestroy } from 'svelte';
  import { GetOpenIncidents, GetResolvedIncidents, GetServicesConfig, SetSelectedServices } from '../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff, BrowserOpenURL, LogDebug } from '../../wailsjs/runtime/runtime.js';

  let activeTab = 'open';
  let incidents = [];
  let servicesConfig = null;
  let selectedServiceIds = [];
  let serviceIdToName = {};
  let showServiceDropdown = false;
  let loading = false;
  let dropdownRef;

  onMount(async () => {
    LogDebug("IncidentsPanel mounted");
    
    // Load services configuration
    await loadServicesConfig();

    // Don't load initial incidents from store - wait for fresh data
    // The backend will fetch fresh data immediately

    // Listen for incident updates
    EventsOn('incidents-updated', (type) => {
      LogDebug(`Incidents update event received: ${type}`);
      if ((type === 'open' && activeTab === 'open') || 
          (type === 'resolved' && activeTab === 'resolved')) {
        loadIncidents();
      }
    });

    // Listen for services config updates
    EventsOn('services-config-updated', async () => {
      await loadServicesConfig();
      loadIncidents();
    });

    // Add click outside handler
    document.addEventListener('click', handleClickOutside);
    
    // Load incidents after services are configured
    loadIncidents();
  });

  onDestroy(() => {
    EventsOff('incidents-updated');
    EventsOff('services-config-updated');
    document.removeEventListener('click', handleClickOutside);
  });

  async function loadServicesConfig() {
    try {
      servicesConfig = await GetServicesConfig();
      LogDebug(`Services config loaded: ${JSON.stringify(servicesConfig)}`);
      
      if (servicesConfig && servicesConfig.services) {
        // Build service ID to name mapping and collect all service IDs
        serviceIdToName = {};
        const allServiceIds = [];
        
        servicesConfig.services.forEach(service => {
          // Handle different ID formats
          if (typeof service.id === 'string') {
            serviceIdToName[service.id] = service.name;
            allServiceIds.push(service.id);
          } else if (Array.isArray(service.id)) {
            service.id.forEach(id => {
              serviceIdToName[id] = service.name;
              allServiceIds.push(id);
            });
          } else if (typeof service.id === 'number') {
            const idStr = service.id.toString();
            serviceIdToName[idStr] = service.name;
            allServiceIds.push(idStr);
          }
        });
        
        // Initialize with all services selected
        selectedServiceIds = [...allServiceIds];
        await updateSelectedServicesBackend();
      } else {
        selectedServiceIds = [];
        serviceIdToName = {};
      }
    } catch (err) {
      LogDebug(`No services configuration found: ${err}`);
      console.log('No services configuration found');
      selectedServiceIds = [];
      serviceIdToName = {};
    }
  }

  async function updateSelectedServicesBackend() {
    try {
      await SetSelectedServices(selectedServiceIds);
      LogDebug(`Updated backend with ${selectedServiceIds.length} selected services`);
    } catch (err) {
      LogDebug(`Failed to update selected services: ${err}`);
    }
  }

  async function loadIncidents() {
    loading = true;
    try {
      if (activeTab === 'open') {
        // Pass selected services to filter on backend
        incidents = await GetOpenIncidents(selectedServiceIds) || [];
      } else {
        // Pass selected services to filter resolved incidents
        incidents = await GetResolvedIncidents(selectedServiceIds) || [];
      }
      LogDebug(`Loaded ${incidents.length} ${activeTab} incidents for ${selectedServiceIds.length} selected services`);
    } catch (err) {
      console.error('Failed to load incidents:', err);
      incidents = [];
    } finally {
      loading = false;
    }
  }

  function handleClickOutside(event) {
    if (dropdownRef && !dropdownRef.contains(event.target)) {
      showServiceDropdown = false;
    }
  }

  function toggleServiceDropdown() {
    showServiceDropdown = !showServiceDropdown;
  }

  function handleDropdownKeyPress(event) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      toggleServiceDropdown();
    }
  }

  async function toggleService(serviceId) {
    const index = selectedServiceIds.indexOf(serviceId);
    if (index > -1) {
      selectedServiceIds = selectedServiceIds.filter(id => id !== serviceId);
    } else {
      selectedServiceIds = [...selectedServiceIds, serviceId];
    }
    
    await updateSelectedServicesBackend();
    // Always reload incidents when services change
    loadIncidents();
  }

  function handleServiceKeyPress(event, serviceId) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      toggleService(serviceId);
    }
  }

  async function selectAllServices() {
    if (servicesConfig && servicesConfig.services) {
      const allServiceIds = [];
      servicesConfig.services.forEach(service => {
        if (typeof service.id === 'string') {
          allServiceIds.push(service.id);
        } else if (Array.isArray(service.id)) {
          allServiceIds.push(...service.id);
        } else if (typeof service.id === 'number') {
          allServiceIds.push(service.id.toString());
        }
      });
      selectedServiceIds = allServiceIds;
      await updateSelectedServicesBackend();
      // Reload incidents with all services
      loadIncidents();
    }
  }

  async function deselectAllServices() {
    selectedServiceIds = [];
    await updateSelectedServicesBackend();
    // Reload incidents with no services
    loadIncidents();
  }

  async function switchTab(tab) {
    activeTab = tab;
    // Load incidents for the new tab with current service filter
    await loadIncidents();
  }

  function handleTabKeyPress(event, tab) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      switchTab(tab);
    }
  }

  function formatDate(dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime(); // Fixed: Convert dates to timestamps
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    
    return date.toLocaleDateString();
  }

  function getStatusColor(status) {
    switch(status) {
      case 'triggered': return '#ff6b6b';
      case 'acknowledged': return '#ffd93d';
      case 'resolved': return '#6bcf7f';
      default: return '#999';
    }
  }

  function getServiceDisplayName(serviceId) {
    return serviceIdToName[serviceId] || 'Unknown Service';
  }

  function getUniqueServices() {
    const uniqueServices = new Map();
    if (servicesConfig && servicesConfig.services) {
      servicesConfig.services.forEach(service => {
        if (typeof service.id === 'string') {
          uniqueServices.set(service.id, service.name);
        } else if (Array.isArray(service.id)) {
          service.id.forEach(id => {
            if (!uniqueServices.has(id)) {
              uniqueServices.set(id, service.name);
            }
          });
        } else if (typeof service.id === 'number') {
          const idStr = service.id.toString();
          uniqueServices.set(idStr, service.name);
        }
      });
    }
    return Array.from(uniqueServices.entries());
  }

  function openIncident(url) {
    BrowserOpenURL(url);
  }

  function handleIncidentKeyPress(event, url) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      openIncident(url);
    }
  }
</script>

<div class="incidents-panel">
  <div class="panel-header">
    <div class="tabs">
      <button 
        class="tab {activeTab === 'open' ? 'active' : ''}" 
        on:click={() => switchTab('open')}
        on:keypress={(e) => handleTabKeyPress(e, 'open')}
        role="tab"
        aria-selected={activeTab === 'open'}>
        Open ({incidents.filter(i => i.status !== 'resolved').length})
      </button>
      <button 
        class="tab {activeTab === 'resolved' ? 'active' : ''}" 
        on:click={() => switchTab('resolved')}
        on:keypress={(e) => handleTabKeyPress(e, 'resolved')}
        role="tab"
        aria-selected={activeTab === 'resolved'}>
        Resolved
      </button>
    </div>
    
    {#if servicesConfig && servicesConfig.services && servicesConfig.services.length > 0}
      <div class="service-filter" bind:this={dropdownRef}>
        <button 
          class="filter-button" 
          on:click={toggleServiceDropdown}
          on:keypress={handleDropdownKeyPress}
          aria-expanded={showServiceDropdown}>
          Services ({selectedServiceIds.length}/{getUniqueServices().length})
          <span class="arrow">{showServiceDropdown ? '▲' : '▼'}</span>
        </button>
        
        {#if showServiceDropdown}
          <div class="dropdown-menu">
            <div class="dropdown-actions">
              <button class="dropdown-action" on:click={selectAllServices}>Select All</button>
              <button class="dropdown-action" on:click={deselectAllServices}>Clear</button>
            </div>
            {#each getUniqueServices() as [serviceId, serviceName]}
              <!-- svelte-ignore a11y-no-noninteractive-element-to-interactive-role -->
              <label 
                class="service-option"
                on:click|stopPropagation
                on:keypress={(e) => handleServiceKeyPress(e, serviceId)}
                tabindex="0"
                role="checkbox"
                aria-checked={selectedServiceIds.includes(serviceId)}>
                <input 
                  type="checkbox" 
                  checked={selectedServiceIds.includes(serviceId)}
                  on:change={() => toggleService(serviceId)}
                />
                <span>{serviceName}</span>
              </label>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <div class="incidents-list">
    {#if loading}
      <div class="loading">Loading incidents...</div>
    {:else if incidents.length === 0}
      <div class="no-incidents">
        {#if activeTab === 'open'}
          No open incidents
        {:else}
          No resolved incidents in the last 7 days
        {/if}
      </div>
    {:else}
      {#each incidents as incident}
        <div 
          class="incident-card" 
          on:click={() => openIncident(incident.html_url)}
          on:keypress={(e) => handleIncidentKeyPress(e, incident.html_url)}
          tabindex="0"
          role="button"
          aria-label="Open incident {incident.incident_number}">
          <div class="incident-header">
            <span class="incident-number">#{incident.incident_number}</span>
            <span class="incident-status" style="color: {getStatusColor(incident.status)}">
              {incident.status}
            </span>
          </div>
          <div class="incident-title">{incident.title}</div>
          <div class="incident-meta">
            <span class="service-name">{getServiceDisplayName(incident.service_id)}</span>
            <span class="incident-time">{formatDate(incident.created_at)}</span>
            {#if incident.alert_count > 0}
              <span class="alert-count">{incident.alert_count} alerts</span>
            {/if}
          </div>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .incidents-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #1a1a1a;
  }

  .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 15px 20px;
    border-bottom: 1px solid #333;
    background: #242424;
  }

  .tabs {
    display: flex;
    gap: 10px;
  }

  .tab {
    background: none;
    border: none;
    color: #999;
    padding: 8px 16px;
    cursor: pointer;
    font-size: 14px;
    border-radius: 20px;
    transition: all 0.3s;
  }

  .tab:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .tab.active {
    background: #007bff;
    color: white;
  }

  .service-filter {
    position: relative;
  }

  .filter-button {
    background: #333;
    border: 1px solid #444;
    color: #fff;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .filter-button:hover {
    background: #444;
  }

  .arrow {
    font-size: 10px;
  }

  .dropdown-menu {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 5px;
    background: #2a2a2a;
    border: 1px solid #444;
    border-radius: 4px;
    min-width: 250px;
    max-height: 300px;
    overflow-y: auto;
    z-index: 100;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
  }

  .dropdown-actions {
    display: flex;
    gap: 10px;
    padding: 10px;
    border-bottom: 1px solid #444;
  }

  .dropdown-action {
    flex: 1;
    background: #007bff;
    border: none;
    color: white;
    padding: 6px 12px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
  }

  .dropdown-action:hover {
    background: #0056b3;
  }

  .service-option {
    display: flex;
    align-items: center;
    padding: 10px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .service-option:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .service-option:focus {
    outline: 2px solid #007bff;
    outline-offset: -2px;
  }

  .service-option input {
    margin-right: 10px;
  }

  .service-option span {
    color: #fff;
    font-size: 14px;
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
    font-size: 16px;
  }

  .incident-card {
    background: #242424;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 15px;
    margin-bottom: 10px;
    cursor: pointer;
    transition: all 0.3s;
  }

  .incident-card:hover {
    background: #2a2a2a;
    border-color: #444;
    transform: translateY(-2px);
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
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
    color: #007bff;
    font-weight: bold;
    font-size: 14px;
  }

  .incident-status {
    font-size: 12px;
    text-transform: uppercase;
    font-weight: bold;
  }

  .incident-title {
    color: #fff;
    font-size: 16px;
    margin-bottom: 10px;
    line-height: 1.4;
  }

  .incident-meta {
    display: flex;
    gap: 15px;
    font-size: 12px;
    color: #999;
  }

  .service-name {
    color: #66b3ff;
  }

  .alert-count {
    color: #ffd93d;
  }

  /* Scrollbar styles */
  .incidents-list::-webkit-scrollbar,
  .dropdown-menu::-webkit-scrollbar {
    width: 8px;
  }

  .incidents-list::-webkit-scrollbar-track,
  .dropdown-menu::-webkit-scrollbar-track {
    background: #1a1a1a;
  }

  .incidents-list::-webkit-scrollbar-thumb,
  .dropdown-menu::-webkit-scrollbar-thumb {
    background: #444;
    border-radius: 4px;
  }

  .incidents-list::-webkit-scrollbar-thumb:hover,
  .dropdown-menu::-webkit-scrollbar-thumb:hover {
    background: #555;
  }
</style>