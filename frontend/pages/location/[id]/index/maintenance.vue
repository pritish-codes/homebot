<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import MaintenanceListView from "~/components/Maintenance/ListView.vue";

  const route = useRoute();
  const locationId = computed<string>(() => route.params.id as string);
</script>

<template>
  <BaseContainer class="flex flex-col gap-8">
    <MaintenanceListView :current-item-id="locationId" />
  </BaseContainer>
</template>
