import { defineConfig } from '@vben/vite-config';

export default defineConfig(async () => {
  return {
    application: {},
    vite: {
      server: {
        proxy: {
          '/api': {
            changeOrigin: true,
            // FCL API berjalan di admin-service port 6905 (lihat docker-compose.yaml)
            target: 'http://localhost:6905',
            ws: true,
          },
        },
      },
    },
  };
});
