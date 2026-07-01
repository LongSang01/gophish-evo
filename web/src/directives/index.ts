import type { App } from 'vue';

// Permission directive
const permission = {
  mounted(_el: HTMLElement, binding: any) {
    const { value } = binding;
    if (value) {
      // Could check against user permissions
      // For now, just pass through
    }
  },
};

export function setupGlobDirectives(app: App) {
  app.directive('permission', permission);
}
