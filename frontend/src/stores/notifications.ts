import { writable, get } from 'svelte/store';
import { GetNotificationConfig, GetAvailableSounds } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

export interface NotificationConfig {
    enabled: boolean;
    sound: string;
    snoozed: boolean;
    snoozeUntil?: Date;
}

export const notificationConfig = writable<NotificationConfig>({
    enabled: true,
    sound: 'default',
    snoozed: false
});

export const availableSounds = writable<string[]>(['default']);

export async function loadNotificationConfig() {
    try {
        const config = await GetNotificationConfig();
        notificationConfig.set(config);
    } catch (err) {
        console.error('Failed to load notification config:', err);
    }
}

export async function loadAvailableSounds() {
    try {
        const sounds = await GetAvailableSounds();
        availableSounds.set(sounds);
    } catch (err) {
        console.error('Failed to load available sounds:', err);
        availableSounds.set(['default']);
    }
}

// Initialize event listeners
export function initializeNotificationListeners() {
    EventsOn('notification-snoozed', (minutes: number) => {
        notificationConfig.update(config => ({
            ...config,
            snoozed: true,
            snoozeUntil: new Date(Date.now() + minutes * 60000)
        }));
    });
    
    EventsOn('notification-unsnoozed', () => {
        notificationConfig.update(config => ({
            ...config,
            snoozed: false,
            snoozeUntil: undefined
        }));
    });
}