interface WorkerEnv {
  API_URL?: string
  ASSETS: { fetch: typeof fetch }
}

export default {
  async fetch(request, env: WorkerEnv) {
    const url = new URL(request.url);

    if (url.pathname.startsWith("/api/")) {
      const apiUrl = new URL(url.pathname + url.search, env.API_URL || 'http://localhost:40080');
      const modifiedRequest = new Request(apiUrl, {
        method: request.method,
        headers: request.headers,
        body: request.body,
      });
      return fetch(modifiedRequest);
    }

    return env.ASSETS.fetch(request);
  },
} satisfies ExportedHandler<WorkerEnv>;
