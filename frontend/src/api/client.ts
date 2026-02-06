import ky from 'ky'

export const api = ky.create({
  prefixUrl: '/api',
  credentials: 'include',
  hooks: {
    afterResponse: [
      async (_request, _options, response) => {
        if (response.status === 401) {
          // Only redirect if not already on login page
          if (!window.location.pathname.includes('/login')) {
            window.location.href = '/login'
          }
        }
      },
    ],
  },
})
