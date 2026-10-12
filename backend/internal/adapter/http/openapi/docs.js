// A file rather than an inline script so the page works under the API's CSP.
window.ui = SwaggerUIBundle({
  url: '/api/openapi.yaml',
  dom_id: '#swagger-ui',
  deepLinking: true,
  withCredentials: true,
})
