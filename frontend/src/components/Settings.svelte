<script lang="ts">
    import { settingsOpen, settingsTab } from '../stores/incidents';
    import SettingsGeneral from './SettingsGeneral.svelte';
    import SettingsService from './SettingsService.svelte';
    
    let errorMessage = '';
    let successMessage = '';
    
    // Clear messages when switching tabs
    $: if ($settingsTab) {
        errorMessage = '';
        successMessage = '';
    }
</script>

{#if $settingsOpen}
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="settings-overlay" on:click={() => settingsOpen.set(false)}></div>
    <div class="settings-panel">
        <div class="settings-header">
            <h2>Settings</h2>
            <p class="settings-subtitle">Configure your PagerOps settings</p>
            <button class="close-button" on:click={() => settingsOpen.set(false)}>
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="6" x2="6" y2="18"></line>
                    <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
            </button>
        </div>
        
        <div class="tabs">
            <button 
                class="tab" 
                class:active={$settingsTab === 'general'}
                on:click={() => settingsTab.set('general')}
            >
                General
            </button>
            <button 
                class="tab" 
                class:active={$settingsTab === 'services'}
                on:click={() => settingsTab.set('services')}
            >
                Service Management
            </button>
        </div>
        
        {#if errorMessage}
            <div class="alert alert-error">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"></circle>
                    <line x1="12" y1="8" x2="12" y2="12"></line>
                    <line x1="12" y1="16" x2="12.01" y2="16"></line>
                </svg>
                {errorMessage}
            </div>
        {/if}
        
        {#if successMessage}
            <div class="alert alert-success">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
                    <polyline points="22 4 12 14.01 9 11.01"></polyline>
                </svg>
                {successMessage}
            </div>
        {/if}
        
        <div class="tab-content">
            {#if $settingsTab === 'general'}
                <SettingsGeneral bind:errorMessage bind:successMessage />
            {:else if $settingsTab === 'services'}
                <SettingsService bind:errorMessage bind:successMessage />
            {/if}
        </div>
    </div>
{/if}

<style>
    .settings-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        backdrop-filter: blur(4px);
        z-index: 999;
        animation: fadeIn 0.2s ease-out;
    }
    
    @keyframes fadeIn {
        from {
            opacity: 0;
        }
        to {
            opacity: 1;
        }
    }
    
    .settings-panel {
        position: fixed;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        background: white;
        border-radius: 12px;
        width: 90%;
        max-width: 560px;
        max-height: 85vh;
        overflow-y: auto;
        z-index: 1000;
        box-shadow: 0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1);
        animation: slideUp 0.3s ease-out;
    }
    
    @keyframes slideUp {
        from {
            opacity: 0;
            transform: translate(-50%, -40%);
        }
        to {
            opacity: 1;
            transform: translate(-50%, -50%);
        }
    }
    
    .settings-header {
        position: relative;
        padding: 24px;
        border-bottom: 1px solid #e5e7eb;
    }
    
    .settings-header h2 {
        margin: 0;
        font-size: 20px;
        font-weight: 600;
        color: #111827;
    }
    
    .settings-subtitle {
        margin: 4px 0 0 0;
        font-size: 13px;
        color: #6b7280;
    }
    
    .close-button {
        position: absolute;
        top: 20px;
        right: 20px;
        width: 32px;
        height: 32px;
        border: none;
        background: #f9fafb;
        border-radius: 8px;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s;
        color: #6b7280;
    }
    
    .close-button:hover {
        background: #f3f4f6;
        color: #374151;
        transform: rotate(90deg);
    }
    
    .tabs {
        display: flex;
        border-bottom: 1px solid #e5e7eb;
        background: #f9fafb;
    }
    
    .tab {
        flex: 1;
        padding: 14px 16px;
        background: transparent;
        border: none;
        cursor: pointer;
        font-size: 14px;
        font-weight: 500;
        color: #6b7280;
        border-bottom: 2px solid transparent;
        transition: all 0.2s;
        position: relative;
    }
    
    .tab:hover {
        color: #374151;
        background: rgba(255, 255, 255, 0.5);
    }
    
    .tab.active {
        color: #3b82f6;
        background: white;
        border-bottom-color: #3b82f6;
    }
    
    .tab-content {
        background: white;
        min-height: 200px;
    }
    
    .alert {
        margin: 16px;
        padding: 12px 16px;
        border-radius: 8px;
        font-size: 14px;
        display: flex;
        align-items: center;
        gap: 10px;
        animation: slideDown 0.3s ease-out;
    }
    
    @keyframes slideDown {
        from {
            opacity: 0;
            transform: translateY(-10px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
    
    .alert svg {
        flex-shrink: 0;
    }
    
    .alert-error {
        background: #fee2e2;
        color: #991b1b;
        border: 1px solid #fecaca;
    }
    
    .alert-error svg {
        color: #dc2626;
    }
    
    .alert-success {
        background: #dcfce7;
        color: #14532d;
        border: 1px solid #bbf7d0;
    }
    
    .alert-success svg {
        color: #16a34a;
    }
    
    /* Custom scrollbar for settings panel */
    .settings-panel::-webkit-scrollbar {
        width: 8px;
    }
    
    .settings-panel::-webkit-scrollbar-track {
        background: #f3f4f6;
        border-radius: 0 8px 8px 0;
    }
    
    .settings-panel::-webkit-scrollbar-thumb {
        background: #d1d5db;
        border-radius: 4px;
    }
    
    .settings-panel::-webkit-scrollbar-thumb:hover {
        background: #9ca3af;
    }
</style>