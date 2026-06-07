import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import Dashboard from '../views/Dashboard.vue'
import Holdings from '../views/Holdings.vue'
import Combos from '../views/Combos.vue'
import ImportExport from '../views/ImportExport.vue'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: Login,
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: Dashboard,
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'DashboardHome',
        redirect: { name: 'Holdings' }
      },
      {
        path: 'holdings',
        name: 'Holdings',
        component: Holdings
      },
      {
        path: 'combos',
        name: 'Combos',
        component: Combos
      },
      {
        path: 'import-export',
        name: 'ImportExport',
        component: ImportExport
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('../views/Users.vue')
      },
      {
        path: 'public-market',
        name: 'PublicMarket',
        component: () => import('../views/PublicMarket.vue')
      }
    ]
  },
  // Fallback redirect
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation Guard
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  const isAuthenticated = !!token

  if (to.meta.requiresAuth && !isAuthenticated) {
    next({ name: 'Login' })
  } else if (to.name === 'Login' && isAuthenticated) {
    next({ name: 'Holdings' })
  } else {
    next()
  }
})

export default router
