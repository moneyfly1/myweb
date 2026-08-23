<template>
  <ClientTutorials :items="androidTutorials" />
</template>

<script setup>
import ClientTutorials from './ClientTutorials.vue'
import { androidTutorials } from './tutorialsData'
</script>
