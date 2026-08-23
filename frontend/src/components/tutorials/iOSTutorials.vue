<template>
  <ClientTutorials :items="iosTutorials" />
</template>

<script setup>
import ClientTutorials from './ClientTutorials.vue'
import { iosTutorials } from './tutorialsData'
</script>
