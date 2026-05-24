import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/connect',
  },
  {
    path: '/connect',
    name: 'connect',
    component: () => import('../pages/ConnectPage.vue'),
  },
  {
    path: '/term',
    name: 'term',
    component: () => import('../pages/TerminalPage.vue'),
  },
  {
    path: '/audit',
    name: 'audit',
    component: () => import('../pages/AuditPage.vue'),
  },
  {
    path: '/playback',
    name: 'playback',
    component: () => import('../pages/PlaybackPage.vue'),
  },
  {
    path: '/rdp',
    name: 'rdp',
    component: () => import('../pages/RdpPage.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
