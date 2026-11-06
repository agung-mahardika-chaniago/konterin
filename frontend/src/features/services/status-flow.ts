/**
 * Client-side mirror of the server state machine (docs/01 §7.1).
 * The API remains the source of truth; this list only drives the UI.
 */
export const ALLOWED_TRANSITIONS: Record<string, string[]> = {
  received: ['waiting_diagnosis', 'cancelled'],
  waiting_diagnosis: ['waiting_approval', 'waiting_part', 'in_progress', 'cancelled'],
  waiting_approval: ['waiting_part', 'in_progress', 'cancelled'],
  waiting_part: ['in_progress', 'cancelled'],
  in_progress: ['waiting_part', 'testing', 'cancelled'],
  testing: ['in_progress', 'completed', 'cancelled'],
  completed: ['ready_for_pickup', 'in_progress', 'cancelled'],
  ready_for_pickup: ['picked_up', 'completed', 'cancelled'],
  picked_up: [],
  cancelled: [],
};

/** Main path shown by the stepper; branch states are handled separately. */
export const MAIN_STATUS_FLOW = [
  'received',
  'waiting_diagnosis',
  'in_progress',
  'testing',
  'completed',
  'ready_for_pickup',
  'picked_up',
] as const;

export const BRANCH_STATUSES = ['waiting_approval', 'waiting_part'] as const;

export const TERMINAL_STATUSES = ['picked_up', 'cancelled'] as const;

export function isTerminalStatus(status: string): boolean {
  return (TERMINAL_STATUSES as readonly string[]).includes(status);
}

export interface TransitionOptions {
  role?: string | null;
  canCancel: boolean;
}

/** Returns the transitions the current actor may perform. */
export function allowedTransitionsFor(status: string, options: TransitionOptions): string[] {
  const targets = ALLOWED_TRANSITIONS[status] ?? [];
  return targets.filter((target) => {
    if (target === 'cancelled' && !options.canCancel) return false;
    // Cashiers only hand devices over (docs/07 RBAC matrix).
    if (options.role === 'cashier' && target !== 'ready_for_pickup' && target !== 'picked_up') {
      return false;
    }
    return true;
  });
}
