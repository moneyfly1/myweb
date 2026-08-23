<template>
  <ClientTutorials :items="macosTutorials" />
</template>

<script setup>
import ClientTutorials from './ClientTutorials.vue'
import { macosTutorials } from './tutorialsData'
</script>
