<script>
  import { ConfigureAPIKey, GetAPIKey, UploadServicesConfig, RemoveServicesConfig, GetServicesConfig } from '../../wailsjs/go/main/App.js';
  import { onMount } from 'svelte';
  import { EventsOn, EventsOff, EventsEmit } from '../../wailsjs/runtime/runtime.js';

  export let isOpen = false;
  
  let activeTab = 'general';
  let apiKey = '';
  let showApiKeyInput = false;
  let servicesConfig = null;
  let fileInput;
  let dragOver = false;

  onMount(() => {
    let mounted = true;
    
    // Initialize data asynchronously
    const initialize = async () => {
      try {
        apiKey = await GetAPIKey();
        servicesConfig = await GetServicesConfig();
      } catch (err) {
        console.log('No API key or services config found');
      }
    };
    
    // Listen for services config updates
    const handleConfigUpdate = async () => {
      if (!mounted) return;
      try {
        servicesConfig = await GetServicesConfig();
      } catch {
        servicesConfig = null;
      }
    };
    
    EventsOn('services-config-updated', handleConfigUpdate);
    
    // Start initialization
    initialize();
    
    // Return cleanup function
    return () => {
      mounted = false;
      EventsOff('services-config-updated');
    };
  });

  async function saveAPIKey() {
    try {
      await ConfigureAPIKey(apiKey);
      showApiKeyInput = false;
      alert('API Key saved successfully');
    } catch (err) {
      alert('Failed to save API Key: ' + err);
    }
  }

  async function removeServicesConfig() {
    if (confirm('Are you sure you want to remove the services configuration?')) {
      try {
        await RemoveServicesConfig();
        servicesConfig = null;
        alert('Services configuration removed');
      } catch (err) {
        alert('Failed to remove services configuration: ' + err);
      }
    }
  }

  async function handleFileSelect(event) {
    const file = event.target.files[0];
    if (file) {
      await processFile(file);
    }
  }

  async function processFile(file) {
  if (file.type !== 'application/json') {
    alert('Please upload a JSON file');
    return;
  }

  const reader = new FileReader();
  reader.onload = async (e) => {
    try {
      const content = e.target.result;
      
      // Ensure content is a string
      if (typeof content !== 'string') {
        alert('Failed to read file content');
        return;
      }
      
      // Parse JSON to validate it and extract service info immediately
      const parsedConfig = JSON.parse(content);
      if (!parsedConfig.services || !Array.isArray(parsedConfig.services)) {
        alert('Invalid services configuration format');
        return;
      }
      
      // Upload to backend
      await UploadServicesConfig(content);
      
      // Update local state immediately
      servicesConfig = parsedConfig;
      
      // The backend will emit 'services-config-updated' event
      // which will trigger the IncidentsPanel to reload
      
      alert(`Services configuration uploaded successfully! ${parsedConfig.services.length} services loaded.`);
    } catch (err) {
      alert('Failed to process services configuration: ' + err);
    }
  };
  
  reader.onerror = () => {
    alert('Failed to read file');
  };
  
  reader.readAsText(file);
}

  function handleDragOver(event) {
    event.preventDefault();
    dragOver = true;
  }

  function handleDragLeave(event) {
    event.preventDefault();
    dragOver = false;
  }

  async function handleDrop(event) {
    event.preventDefault();
    dragOver = false;
    
    const files = event.dataTransfer.files;
    if (files.length > 0) {
      await processFile(files[0]);
    }
  }

  function closeModal() {
    isOpen = false;
  }

  function handleKeyDown(event) {
    if (event.key === 'Escape') {
      closeModal();
    }
  }
</script>

{#if isOpen}
<div class="modal-overlay" on:click={closeModal} on:keydown={handleKeyDown}>
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="modal-content" on:click|stopPropagation>
    <div class="modal-header">
      <h2>Settings</h2>
      <button class="close-button" on:click={closeModal} type="button">
        ×
      </button>
    </div>
    
    <div class="tabs">
      <button 
        class="tab {activeTab === 'general' ? 'active' : ''}" 
        on:click={() => activeTab = 'general'}
        type="button">
        General
      </button>
      <button 
        class="tab {activeTab === 'services' ? 'active' : ''}" 
        on:click={() => activeTab = 'services'}
        type="button">
        Services
      </button>
    </div>
    
    <div class="tab-content">
      {#if activeTab === 'general'}
        <div class="settings-section">
          <h3>PagerDuty API Configuration</h3>
          
          <div class="setting-item">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label>API Key Status</label>
            {#if apiKey}
              <p class="api-status">✓ API Key configured</p>
              <button class="change-btn" on:click={() => showApiKeyInput = !showApiKeyInput} type="button">
                Change API Key
              </button>
            {:else}
              <p style="color: #ff6b6b;">No API Key configured</p>
              <button class="change-btn" on:click={() => showApiKeyInput = true} type="button">
                Add API Key
              </button>
            {/if}
            
            {#if showApiKeyInput}
              <div class="api-key-input">
                <input 
                  type="password" 
                  bind:value={apiKey} 
                  placeholder="Enter your PagerDuty API key"
                />
                <button class="save-btn" on:click={saveAPIKey} type="button">
                  Save API Key
                </button>
              </div>
            {/if}
          </div>
        </div>
      {/if}
      
      {#if activeTab === 'services'}
        <div class="settings-section">
          <h3>Services Configuration</h3>
          
          {#if servicesConfig}
            <button class="remove-btn" on:click={removeServicesConfig} type="button">
              Remove Current Configuration
            </button>
          {/if}
          
          <div 
            class="drop-zone {dragOver ? 'drag-over' : ''}"
            on:drop={handleDrop}
            on:dragover={handleDragOver}
            on:dragleave={handleDragLeave}
            role="region"
            aria-label="File drop zone"
          >
            <input 
              type="file" 
              accept=".json" 
              bind:this={fileInput}
              on:change={handleFileSelect}
              style="display: none"
            />
            <p>{servicesConfig ? 'Replace configuration' : 'Drop JSON file here or'}</p>
            <button 
              class="upload-btn" 
              on:click={() => fileInput.click()}
              type="button">
              Choose File
            </button>
          </div>
          
          {#if servicesConfig?.services}
            <div class="services-list">
              <h4>Configured Services ({servicesConfig.services.length}):</h4>
              {#each servicesConfig.services as service}
                <div class="service-item">
                  <span class="service-name">{service.name}</span>
                  <span class="service-ids">
                    {#if typeof service.id === 'string'}
                      ID: {service.id}
                    {:else if Array.isArray(service.id)}
                      IDs: {service.id.join(', ')}
                    {:else}
                      ID: {service.id}
                    {/if}
                  </span>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
</div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.8);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
  }

  .modal-content {
    background: #2a2a2a;
    border-radius: 8px;
    width: 90%;
    max-width: 600px;
    max-height: 80vh;
    overflow-y: auto;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px;
    border-bottom: 1px solid #444;
  }

  .modal-header h2 {
    margin: 0;
    color: #fff;
  }

  .close-button {
    background: none;
    border: none;
    color: #999;
    font-size: 24px;
    cursor: pointer;
    padding: 0;
    width: 30px;
    height: 30px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .close-button:hover {
    color: #fff;
  }

  .tabs {
    display: flex;
    border-bottom: 1px solid #444;
    padding: 0 20px;
  }

  .tab {
    background: none;
    border: none;
    color: #999;
    padding: 15px 20px;
    cursor: pointer;
    font-size: 14px;
    border-bottom: 2px solid transparent;
    transition: all 0.3s;
  }

  .tab:hover {
    color: #fff;
  }

  .tab.active {
    color: #fff;
    border-bottom-color: #007bff;
  }

  .tab-content {
    padding: 20px;
  }

  .settings-section {
    margin-bottom: 30px;
  }

  .settings-section h3 {
    color: #fff;
    margin-bottom: 20px;
  }

  .setting-item {
    margin-bottom: 20px;
  }

  .setting-item label {
    display: block;
    color: #999;
    margin-bottom: 10px;
    font-size: 14px;
  }

  .change-btn, .upload-btn, .save-btn {
    background: #007bff;
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
    margin-top: 10px;
  }

  .change-btn:hover, .upload-btn:hover, .save-btn:hover {
    background: #0056b3;
  }

  .remove-btn {
    background: #dc3545;
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
    margin-bottom: 20px;
  }

  .remove-btn:hover {
    background: #c82333;
  }

  .api-status {
    margin-top: 10px;
    color: #4caf50;
  }

  .api-key-input {
    margin-top: 15px;
  }

  .api-key-input input {
    width: 100%;
    padding: 10px;
    border: 1px solid #444;
    border-radius: 4px;
    background: #1a1a1a;
    color: #fff;
    margin-bottom: 10px;
  }

  .drop-zone {
    border: 2px dashed #444;
    border-radius: 8px;
    padding: 40px;
    text-align: center;
    transition: all 0.3s;
    margin-top: 20px;
  }

  .drop-zone.drag-over {
    border-color: #007bff;
    background: rgba(0, 123, 255, 0.1);
  }

  .drop-zone p {
    color: #999;
    margin-bottom: 15px;
  }

  .services-list {
    margin-top: 30px;
    border-top: 1px solid #444;
    padding-top: 20px;
  }

  .services-list h4 {
    color: #fff;
    margin-bottom: 15px;
  }

  .service-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px;
    background: #1a1a1a;
    border-radius: 6px;
    margin-bottom: 8px;
    border: 1px solid rgba(255, 255, 255, 0.06);
  }

  .service-name {
    color: #fff;
    font-weight: 500;
    font-size: 14px;
  }

  .service-ids {
    color: #6b7280;
    font-size: 12px;
    font-family: monospace;
  }

  /* Custom scrollbar for modal */
  .modal-content::-webkit-scrollbar {
    width: 6px;
  }

  .modal-content::-webkit-scrollbar-track {
    background: transparent;
  }

  .modal-content::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.1);
    border-radius: 3px;
  }

  .modal-content::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.15);
  }
</style>