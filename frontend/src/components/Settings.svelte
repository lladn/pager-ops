<script>
  import { ConfigureAPIKey, GetAPIKey, UploadServicesConfig, GetServicesConfig } from '../../wailsjs/go/main/App.js';
  import { onMount } from 'svelte';

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
        // Ensure content is a string
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
    // Only close if clicking the overlay, not the modal content
    if (event && event.target && event.target.classList && event.target.classList.contains('modal-overlay')) {
      isOpen = false;
    }
  }

  function closeSettings() {
    console.log('Closing settings modal');
    isOpen = false;
  }

  function handleKeyDown(event) {
    if (event.key === 'Escape') {
      isOpen = false;
    }
  }
</script>

{#if isOpen}
<!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
<div 
  class="modal-overlay" 
  on:click={closeModal}
  on:keydown={handleKeyDown}
  role="dialog"
  aria-modal="true"
  aria-labelledby="settings-title"
  style="--wails-draggable: no-drag;">
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="modal-panel" on:click|stopPropagation style="--wails-draggable: no-drag;">
    <div class="modal-header">
      <h2 id="settings-title">Settings</h2>
      <button 
        class="close-btn" 
        on:click={closeSettings}
        on:mousedown|preventDefault|stopPropagation
        on:mouseup|preventDefault|stopPropagation
        type="button"
        aria-label="Close settings"
        style="--wails-draggable: no-drag;">×</button>
    </div>
    
    <div class="tabs">
      <button 
        class="tab {activeTab === 'general' ? 'active' : ''}" 
        on:click={() => activeTab = 'general'}
        on:mousedown|preventDefault|stopPropagation
        on:mouseup|preventDefault|stopPropagation
        type="button"
        style="--wails-draggable: no-drag;">
        General
      </button>
      <button 
        class="tab {activeTab === 'configuration' ? 'active' : ''}" 
        on:click={() => activeTab = 'configuration'}
        on:mousedown|preventDefault|stopPropagation
        on:mouseup|preventDefault|stopPropagation
        type="button"
        style="--wails-draggable: no-drag;">
        Configuration
      </button>
    </div>

    <div class="tab-content">
      {#if activeTab === 'general'}
        <div class="general-tab">
          <h3>API Key Configuration</h3>
          {#if !showApiKeyInput}
            <button 
              class="config-btn" 
              on:click={() => showApiKeyInput = true}
              on:mousedown|preventDefault|stopPropagation
              on:mouseup|preventDefault|stopPropagation
              type="button"
              style="--wails-draggable: no-drag;">
              Configure API Key
            </button>
            {#if apiKey}
              <p class="api-status">✓ API Key configured</p>
            {/if}
          {:else}
            <div class="api-key-input">
              <input 
                type="password" 
                bind:value={apiKey} 
                placeholder="Enter PagerDuty API Key"
              />
              <div class="button-group">
                <button 
                  class="save-btn" 
                  on:click={saveAPIKey}
                  on:mousedown|preventDefault|stopPropagation
                  on:mouseup|preventDefault|stopPropagation
                  type="button"
                  style="--wails-draggable: no-drag;">Save</button>
                <button 
                  class="cancel-btn" 
                  on:click={() => showApiKeyInput = false}
                  on:mousedown|preventDefault|stopPropagation
                  on:mouseup|preventDefault|stopPropagation
                  type="button"
                  style="--wails-draggable: no-drag;">Cancel</button>
              </div>
            </div>
          {/if}
        </div>
      {:else if activeTab === 'configuration'}
        <div class="configuration-tab">
          <h3>Services Configuration</h3>
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
            <p>Drop JSON file here or</p>
            <button 
              class="upload-btn" 
              on:click={() => fileInput.click()}
              on:mousedown|preventDefault|stopPropagation
              on:mouseup|preventDefault|stopPropagation
              type="button"
              style="--wails-draggable: no-drag;">
              Choose File
            </button>
          </div>
          
          {#if servicesConfig && servicesConfig.services}
            <div class="services-list">
              <h4>Configured Services:</h4>
              {#each servicesConfig.services as service}
                <div class="service-item">
                  <span class="service-name">{service.name}</span>
                  <span class="service-ids">
                    {#if typeof service.id === 'string'}
                      {service.id}
                    {:else}
                      {service.id.join(', ')}
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
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
    backdrop-filter: blur(2px);
  }

  .modal-panel {
    background: #2a2a2a;
    border-radius: 8px;
    width: 90%;
    max-width: 600px;
    max-height: 80vh;
    overflow-y: auto;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
    position: relative;
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

  .close-btn {
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

  .close-btn:hover {
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
    border-bottom: 2px solid transparent;
    transition: all 0.3s;
    font-size: 14px;
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

  h3 {
    margin-top: 0;
    margin-bottom: 20px;
    color: #fff;
  }

  .config-btn, .upload-btn {
    background: #007bff;
    color: white;
    border: none;
    padding: 10px 20px;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.3s;
    font-size: 14px;
  }

  .config-btn:hover, .upload-btn:hover {
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
  }
</style>