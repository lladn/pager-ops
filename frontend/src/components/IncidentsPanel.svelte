<script>
  import { onMount, onDestroy } from 'svelte';
  import { GetOpenIncidents, GetResolvedIncidents, SetSelectedServices, GetServicesConfig } from '../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff, BrowserOpenURL } from '../../wailsjs/runtime/runtime.js';
  
  export let servicesConfig = null;
  export let selectedServiceIds = [];

  let incidents = [];
  let activeTab = 'open';
  let loading = false;
  let showServiceDropdown = false;
  let dropdownRef;
  let serviceIdToName = {};

  function updateServiceIdToName() {
    serviceIdToName = {};
    if (servicesConfig?.services) {
      servicesConfig.services.forEach(service => {
        if (typeof service.id === 'string') {
          serviceIdToName[service.id] = service.name;
        } else if (Array.isArray(service.id)) {
          service.id.forEach(id => {
            serviceIdToName[id] = service.name;
          });
        } else if (typeof service.id === 'number') {
          serviceIdToName[service.id.toString()] = service.name;
        }
      });
    }
  }

  async function loadIncidents(showLoading = true) {
    if (showLoading) {
      loading = true;
    }
    try {
      if (activeTab === 'open') {
        incidents = await GetOpenIncidents(selectedServiceIds);
      } else {
        incidents = await GetResolvedIncidents(selectedServiceIds);
      }
    } catch (error) {
      console.error('Failed to load incidents:', error);
      incidents = [];
    } finally {
      if (showLoading) {
        loading = false;
      }
    }
  }

  async function updateSelectedServicesBackend() {
    try {
      await SetSelectedServices(selectedServiceIds);
    } catch (error) {
      console.error('Failed to update selected services:', error);
    }
  }

  onMount(async () => {
  // Wait for runtime to be ready
  await new Promise(resolve => setTimeout(resolve, 100));
  
  try {
    // Load services config first
    servicesConfig = await GetServicesConfig();
    updateServiceIdToName();
    
    // Then load incidents
    await loadIncidents();
  } catch (error) {
    console.error('Failed to initialize:', error);
  }
  
  // Set up event listeners after initialization
  EventsOn('incidents-updated', handleIncidentsUpdated);
  EventsOn('services-config-updated', handleServicesConfigUpdated);
  
  // Cleanup function
  return () => {
    EventsOff('incidents-updated');
    EventsOff('services-config-updated');
  };
});
  
  async function loadServicesConfig() {
    try {
      servicesConfig = await GetServicesConfig();
      updateServiceIdToName();
      if (servicesConfig?.services) {
        selectedServiceIds = getUniqueServices().map(([id]) => id);
        await updateSelectedServicesBackend();
      }
    } catch (err) {
      console.log('No services config found');
      servicesConfig = null;
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
    loadIncidents(true); // Show loading when filter changes
  }

  function handleServiceKeyPress(event, serviceId) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      toggleService(serviceId);
    }
  }

  async function selectAllServices() {
    if (servicesConfig?.services) {
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
      loadIncidents(true); // Show loading when filter changes
    }
  }

  async function deselectAllServices() {
    selectedServiceIds = [];
    await updateSelectedServicesBackend();
    loadIncidents(true); // Show loading when filter changes
  }

  async function switchTab(tab) {
    activeTab = tab;
    await loadIncidents(true); // Show loading when switching tabs
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
    const diffMs = now.getTime() - date.getTime();
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
      case 'triggered': return '#ef4444';
      case 'acknowledged': return '#f59e0b';
      case 'resolved': return '#10b981';
      default: return '#6b7280';
    }
  }

  function getServiceDisplayName(serviceId) {
    return serviceIdToName[serviceId] || 'Unknown Service';
  }

  function getUniqueServices() {
    const uniqueServices = new Map();
    if (servicesConfig?.services) {
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
        <span class="tab-label">Open</span>
        <span class="tab-count">{incidents.filter(i => i.status !== 'resolved').length}</span>
      </button>
      <button 
        class="tab {activeTab === 'resolved' ? 'active' : ''}" 
        on:click={() => switchTab('resolved')}
        on:keypress={(e) => handleTabKeyPress(e, 'resolved')}
        role="tab"
        aria-selected={activeTab === 'resolved'}>
        <span class="tab-label">Resolved</span>
      </button>
    </div>
    
    {#if servicesConfig?.services?.length > 0}
      <div class="service-filter" bind:this={dropdownRef}>
        <button 
          class="filter-button" 
          on:click={toggleServiceDropdown}
          on:keypress={handleDropdownKeyPress}
          aria-expanded={showServiceDropdown}>
          <svg class="filter-icon" width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M2 4H14M4 8H12M6 12H10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          <span class="filter-text">
            {selectedServiceIds.length === getUniqueServices().length 
              ? 'All services' 
              : `${selectedServiceIds.length} services`}
          </span>
          <svg class="dropdown-arrow {showServiceDropdown ? 'rotated' : ''}" width="12" height="12" viewBox="0 0 12 12" fill="none">
            <path d="M3 4.5L6 7.5L9 4.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </button>
        
        {#if showServiceDropdown}
          <div class="dropdown-menu">
            <div class="dropdown-actions">
              <button class="dropdown-action" on:click={selectAllServices}>All</button>
              <button class="dropdown-action" on:click={deselectAllServices}>None</button>
            </div>
            <div class="dropdown-divider"></div>
            <div class="dropdown-options">
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
                    class="service-checkbox"
                  />
                  <span class="service-name">{serviceName}</span>
                </label>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <div class="incidents-list">
    {#if loading}
      <div class="empty-state">
        <div class="loading-spinner"></div>
        <span class="loading-text">Loading incidents...</span>
      </div>
    {:else if incidents.length === 0}
      <div class="empty-state">
        <svg class="empty-icon" width="48" height="48" viewBox="0 0 48 48" fill="none">
          <circle cx="24" cy="24" r="20" stroke="currentColor" stroke-width="2" opacity="0.2"/>
          <path d="M16 20C16 20 20 24 24 24C28 24 32 20 32 20" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.4"/>
          <circle cx="18" cy="18" r="2" fill="currentColor" opacity="0.4"/>
          <circle cx="30" cy="18" r="2" fill="currentColor" opacity="0.4"/>
        </svg>
        <p class="empty-title">
          {#if activeTab === 'open'}
            No open incidents
          {:else}
            No resolved incidents
          {/if}
        </p>
        <p class="empty-subtitle">
          {#if activeTab === 'open'}
            All systems operational
          {:else}
            No incidents resolved in the last 7 days
          {/if}
        </p>
      </div>
    {:else}
      <div class="incidents-container">
        {#each incidents as incident}
          <div 
            class="incident-card" 
            on:click={() => openIncident(incident.html_url)}
            on:keypress={(e) => handleIncidentKeyPress(e, incident.html_url)}
            tabindex="0"
            role="button"
            aria-label="Open incident {incident.incident_number}">
            
            <div class="incident-status-indicator" style="background-color: {getStatusColor(incident.status)}"></div>
            
            <div class="incident-content">
              <div class="incident-header">
                <div class="incident-id">
                  <span class="incident-number">#{incident.incident_number}</span>
                  <span class="incident-badge" style="color: {getStatusColor(incident.status)}">
                    {incident.status}
                  </span>
                </div>
                <span class="incident-time">{formatDate(incident.created_at)}</span>
              </div>
              
              <h3 class="incident-title">{incident.title}</h3>
              
              <div class="incident-footer">
                <span class="incident-service">{getServiceDisplayName(incident.service_id)}</span>
                {#if incident.alert_count > 0}
                  <div class="incident-alerts">
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                      <path d="M7 2.5V7L9.5 9.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                      <circle cx="7" cy="7" r="5" stroke="currentColor" stroke-width="1.5"/>
                    </svg>
                    <span>{incident.alert_count}</span>
                  </div>
                {/if}
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .incidents-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #0a0a0a;
  }

  .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px 24px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    background: rgba(255, 255, 255, 0.01);
  }

  .tabs {
    display: flex;
    gap: 4px;
  }

  .tab {
    background: transparent;
    border: none;
    color: #6b7280;
    padding: 8px 16px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    border-radius: 8px;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .tab:hover {
    background: rgba(255, 255, 255, 0.04);
  }

  .tab.active {
    background: rgba(255, 255, 255, 0.08);
    color: #ffffff;
  }

  .tab-label {
    font-weight: 500;
  }

  .tab-count {
    background: rgba(255, 255, 255, 0.1);
    padding: 2px 6px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 600;
  }

  .service-filter {
    position: relative;
  }

  .filter-button {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 8px;
    color: #e5e7eb;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .filter-button:hover {
    background: rgba(255, 255, 255, 0.06);
    border-color: rgba(255, 255, 255, 0.12);
  }

  .filter-icon {
    color: #9ca3af;
  }

  .filter-text {
    color: #e5e7eb;
  }

  .dropdown-arrow {
    color: #9ca3af;
    transition: transform 0.2s ease;
  }

  .dropdown-arrow.rotated {
    transform: rotate(180deg);
  }

  .dropdown-menu {
    position: absolute;
    top: calc(100% + 8px);
    right: 0;
    min-width: 240px;
    background: #141414;
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 12px;
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
    z-index: 1000;
    overflow: hidden;
  }

  .dropdown-actions {
    display: flex;
    gap: 8px;
    padding: 12px;
  }

  .dropdown-action {
    flex: 1;
    padding: 6px 12px;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 6px;
    color: #e5e7eb;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .dropdown-action:hover {
    background: rgba(255, 255, 255, 0.08);
    border-color: rgba(255, 255, 255, 0.12);
  }

  .dropdown-divider {
    height: 1px;
    background: rgba(255, 255, 255, 0.06);
  }

  .dropdown-options {
    max-height: 300px;
    overflow-y: auto;
    padding: 8px;
  }

  .service-option {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.2s ease;
  }

  .service-option:hover {
    background: rgba(255, 255, 255, 0.04);
  }

  .service-checkbox {
    width: 16px;
    height: 16px;
    accent-color: #3b82f6;
    cursor: pointer;
  }

  .service-name {
    color: #e5e7eb;
    font-size: 13px;
  }

  .incidents-list {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #6b7280;
    text-align: center;
    padding: 48px;
  }

  .loading-spinner {
    width: 32px;
    height: 32px;
    border: 3px solid rgba(255, 255, 255, 0.1);
    border-top-color: #3b82f6;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .loading-text {
    margin-top: 16px;
    font-size: 14px;
    color: #9ca3af;
  }

  .empty-icon {
    color: #374151;
    margin-bottom: 16px;
  }

  .empty-title {
    font-size: 16px;
    font-weight: 600;
    color: #9ca3af;
    margin: 0 0 8px 0;
  }

  .empty-subtitle {
    font-size: 14px;
    color: #6b7280;
    margin: 0;
  }

  .incidents-container {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .incident-card {
    display: flex;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 12px;
    overflow: hidden;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .incident-card:hover {
    background: rgba(255, 255, 255, 0.04);
    border-color: rgba(255, 255, 255, 0.1);
    transform: translateX(4px);
  }

  .incident-card:focus {
    outline: 2px solid #3b82f6;
    outline-offset: 2px;
  }

  .incident-status-indicator {
    width: 4px;
    flex-shrink: 0;
  }

  .incident-content {
    flex: 1;
    padding: 16px 20px;
  }

  .incident-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .incident-id {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .incident-number {
    color: #3b82f6;
    font-size: 13px;
    font-weight: 600;
  }

  .incident-badge {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .incident-time {
    color: #6b7280;
    font-size: 12px;
  }

  .incident-title {
    color: #e5e7eb;
    font-size: 14px;
    font-weight: 500;
    line-height: 1.5;
    margin: 0 0 12px 0;
  }

  .incident-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .incident-service {
    color: #60a5fa;
    font-size: 12px;
    font-weight: 500;
  }

  .incident-alerts {
    display: flex;
    align-items: center;
    gap: 4px;
    color: #f59e0b;
    font-size: 12px;
    font-weight: 500;
  }

  /* Custom scrollbar */
  .incidents-list::-webkit-scrollbar,
  .dropdown-options::-webkit-scrollbar {
    width: 6px;
  }

  .incidents-list::-webkit-scrollbar-track,
  .dropdown-options::-webkit-scrollbar-track {
    background: transparent;
  }

  .incidents-list::-webkit-scrollbar-thumb,
  .dropdown-options::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.1);
    border-radius: 3px;
  }

  .incidents-list::-webkit-scrollbar-thumb:hover,
  .dropdown-options::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.15);
  }
</style>