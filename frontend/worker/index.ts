interface WorkerEnv {
  API_URL: string
  COMMIT_HASH?: string
  ASSETS?: { fetch: typeof fetch }
}

export default {
  async fetch(request, env: WorkerEnv) {
    const url = new URL(request.url);

    if (url.pathname.startsWith("/api/")) {
      if (!env.API_URL) {
        return new Response(JSON.stringify({ error: 'API_URL not configured' }), {
          status: 500,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      const backendPath = url.pathname.replace(/^\/api/, '') + url.search;
      const apiUrl = new URL(backendPath, env.API_URL);
      const modifiedRequest = new Request(apiUrl, {
        method: request.method,
        headers: request.headers,
        body: request.body,
      });
      return fetch(modifiedRequest);
    }

    let res: Response;
    if (env.ASSETS) {
      res = await env.ASSETS.fetch(request);
    } else {
      res = new Response('Not found', { status: 404 });
    }
    // Clone response to add custom headers (SPA HTML and API responses)
    const modified = new Response(res.body, {
      status: res.status,
      statusText: res.statusText,
      headers: res.headers,
    });
    modified.headers.set('X-Environment', env.API_URL ? 'production' : 'development');
    if (env.COMMIT_HASH) {
      modified.headers.set('X-Commit-Hash', env.COMMIT_HASH);
    }
    return modified;
  },
} satisfies ExportedHandler<WorkerEnv>;
