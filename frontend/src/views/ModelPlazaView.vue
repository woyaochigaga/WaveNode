<template>
  <!-- 首页与控制台共用独立广场页面，不再根据入口切换后台侧栏布局。 -->
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <PlazaNavBar />
    <main class="mx-auto max-w-[1440px] px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
      <ModelPlazaContent :response="data" :loading="loading" :error="loadFailed" @retry="loadModelPlaza" />
    </main>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import ModelPlazaContent from '@/components/modelPlaza/ModelPlazaContent.vue'
import { getModelPlaza, type ModelPlazaResponse } from '@/api/modelPlaza'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const data = ref<ModelPlazaResponse | null>(null)
const loading = ref(true)
const loadFailed = ref(false)

async function loadModelPlaza() {
  loading.value = true
  loadFailed.value = false

  try {
    data.value = await getModelPlaza()
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  // 广场导航条需要站点名；有 __APP_CONFIG__ 注入时同步命中缓存。
  void appStore.fetchPublicSettings()
  void loadModelPlaza()
})
</script>
