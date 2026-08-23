<template>
  <ClientTutorials :items="windowsTutorials" />
</template>

<script setup>
import ClientTutorials from './ClientTutorials.vue'
import { windowsTutorials } from './tutorialsData'
</script>
