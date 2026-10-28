// jsdom has no matchMedia, which prime-core reads when it loads.
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
  }),
});

// jsdom has no ResizeObserver, which LayerCake uses to measure its container.
class ResizeObserverStub {
  observe(): void {
    return undefined;
  }
  unobserve(): void {
    return undefined;
  }
  disconnect(): void {
    return undefined;
  }
}
Object.defineProperty(window, 'ResizeObserver', { writable: true, value: ResizeObserverStub });
