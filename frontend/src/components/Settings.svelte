<script>
  import { ConfigureAPIKey, GetAPIKey, UploadServicesConfig, RemoveServicesConfig, GetServicesConfig } from '../../wailsjs/go/main/App.js';
  import { onMount } from 'svelte';
  import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';

  export let isOpen = false;
  
  let activeTab = 'general';
  let apiKey = '';
  let showApiKeyInput = false;
  let servicesConfig = null;
  let fileInput;
  let dragOver = false;

  onMount(async () => {
    try {
      apiKey = await GetAPIKey();
      servicesConfig = await GetServicesConfig();
    } catch (err) {
      console.log('No API key or services config found');
    }

    // Listen for services config updates
    EventsOn('services-config-updated', async () => {
      try {
        servicesConfig = await GetServicesConfig();
      } catch {
        servicesConfig = null;
      }
    });

    return () => {
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
        alert('Services configuration removed successfully');
      } catch (err) {
        alert('Failed to remove configuration: ' + err);
      }
    }
  }

  function handleFileSelect(event) {
    const file = event.target.files[0];
    if (file) {
      readFile(file);
    }
  }

  function handleDrop(event) {
    event.preventDefault();
    dragOver = false;
    const file = event.dataTransfer.files[0];
    if (file && file.type === 'application/json') {
      readFile(file);
    }
  }

  function readFile(file) {
    const reader = new FileReader();
    reader.onload = async (e) => {
      try {
        const content = e.target.result;
        if (typeof content === 'string') {
          await UploadServicesConfig(content);
          servicesConfig = JSON.parse(content);
          alert('Services configuration uploaded successfully');
        } else {
          alert('Invalid file format');
        }
      } catch (err) {
        alert('Failed to upload configuration: ' + err);
      }
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

  function closeModal(event) {
    if (event && event.target && event.target.classList && event.target.classList.contains('modal-overlay')) {
      isOpen = false;
    }
  }

  function closeSettings() {
    isOpen = false;
  }

  function handleTabKeyPress(event, tabName) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      activeTab = tabName;
    }
  }

  function handleCloseKeyPress(event) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      closeSettings();
    }
  }
</script>

{#if isOpen}
<div class="modal-overlay" 
     on:click={closeModal}
     on:keypress={(e) => e.key === 'Escape' && closeSettings()}
     role="dialog"
     aria-modal="true"
     aria-label="Settings">
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="modal-content" on:click|stopPropagation>
    <div class="modal-header">
      <h2>Settings</h2>
      <button class="close-button" 
              on:click={closeSettings}
              on:keypress={handleCloseKeyPress}
              aria-label="Close settings">×</button>
    </div>
    
    <div class="tabs">
      <button 
        class="tab {activeTab === 'general' ? 'active' : ''}" 
        on:click={() => activeTab = 'general'}
        on:keypress={(e) => handleTabKeyPress(e, 'general')}
        role="tab"
        aria-selected={activeTab === 'general'}>
        General
      </button>
      <button 
        class="tab {activeTab === 'services' ? 'active' : ''}" 
        on:click={() => activeTab = 'services'}
        on:keypress={(e) => handleTabKeyPress(e, 'services')}
        role="tab"
        aria-selected={activeTab === 'services'}>
        Services
      </button>
    </div>
    
    <div class="tab-content">
      {#if activeTab === 'general'}
        <div class="settings-section">
          <h3>PagerDuty Configuration</h3>
          <div class="setting-item">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label>API Key</label>
            {#if !showApiKeyInput}
              <div class="api-status">✓ Configured</div>
              <button class="change-btn" on:click={() => showApiKeyInput = true}>
                Change API Key
              </button>
            {:else}
              <div class="api-key-input">
                <input 
                  type="password" 
                  bind:value={apiKey} 
                  placeholder="Enter your PagerDuty API key"
                />
                <div class="button-group">
                  <button class="save-btn" on:click={saveAPIKey}>Save</button>
                  <button class="cancel-btn" on:click={() => showApiKeyInput = false}>Cancel</button>
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/if}
      
      {#if activeTab === 'services'}
        <div class="settings-section">
          <h3>Services Configuration</h3>
          
          {#if servicesConfig && servicesConfig.services}
            <div class="config-actions">
              <button class="remove-config-btn" on:click={removeServicesConfig}>
                Remove Configuration
              </button>
            </div>
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
          
          {#if servicesConfig && servicesConfig.services}
            <div class="services-list">
              <h4>Configured Services ({servicesConfig.services.length}):</h4>
              {#each servicesConfig.services as service}
                <div class="service-item">
                  <span class="service-name">{service.name}</span>
                  <span class="service-ids">
                    {#if typeof service.id === 'string'}
                      {service.id}
                    {:else if Array.isArray(service.id)}
                      {service.id.join(', ')}
                    {:else}
                      {service.id}
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

  .change-btn, .upload-btn {
    background: #007bff;
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
  }

  .change-btn:hover, .upload-btn:hover {
    background: #0056b3;
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
    box-sizing: border-box;
  }

  .button-group {
    display: flex;
    gap: 10px;
  }

  .save-btn {
    background: #4caf50;
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
  }

  .save-btn:hover {
    background: #45a049;
  }

  .cancel-btn {
    background: #666;
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
  }

  .cancel-btn:hover {
    background: #555;
  }

  .config-actions {
    margin-bottom: 20px;
  }

  .remove-config-btn {
    background: #dc3545;
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
  }

  .remove-config-btn:hover {
    background: #c82333;
  }

  .drop-zone {
    border: 2px dashed #666;
    border-radius: 8px;
    padding: 40px;
    text-align: center;
    transition: all 0.3s;
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
    padding: 10px;
    background: #1a1a1a;
    border-radius: 4px;
    margin-bottom: 8px;
  }

  .service-name {
    color: #fff;
    font-weight: bold;
  }

  .service-ids {
    color: #999;
    font-size: 12px;
    font-family: monospace;
  }
</style>