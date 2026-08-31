// Playwright 测试配置：使用独立端口和临时 SQLite，避免污染本地开发数据。
import { defineConfig, devices } from '@playwright/test'
import { mkdtempSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'

const backendPort = 8082
const frontendPort = 5174
const databaseDirectory = mkdtempSync(join(tmpdir(), 'idle-e2e-'))
const databasePath = resolve(databaseDirectory, 'idle.db')

export default defineConfig({
  testDir: './e2e',
  timeout: 45_000,
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL: `http://127.0.0.1:${frontendPort}`,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  webServer: [
    {
      command: 'go run .',
      cwd: '../backend',
      url: `http://127.0.0.1:${backendPort}/api/health`,
      timeout: 120_000,
      reuseExistingServer: false,
      env: {
        ...process.env,
        APP_ENV: 'test',
        HTTP_ADDR: `:${backendPort}`,
        DATABASE_DRIVER: 'sqlite',
        DATABASE_URL: databasePath,
        ALLOWED_ORIGINS: `http://127.0.0.1:${frontendPort}`,
        MIGRATE_ON_START: 'true',
        SEED_ON_START: 'true',
      },
    },
    {
      command: 'npm.cmd run dev -- --host 127.0.0.1 --port 5174',
      cwd: '.',
      url: `http://127.0.0.1:${frontendPort}`,
      timeout: 120_000,
      reuseExistingServer: false,
      env: {
        ...process.env,
        VITE_API_TARGET: `http://127.0.0.1:${backendPort}`,
      },
    },
  ],
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
