<template>
  <div class="sidebar-logo-container"
       :class="{'collapse':collapse}">
    <router-link class="sidebar-logo-link"
                 to="/">
      <span class="logo-text">GVP</span>
    </router-link>
  </div>
</template>

<script>

export default {
  name: 'SidebarLogo',
  props: {
    collapse: {
      type: Boolean,
      required: true
    }
  }
}
</script>

<style lang="scss">
  @import "~@/styles/common/variables";

  .sidebar-logo-container {
    position: relative;
    height: $header-height;
    line-height: $header-height;
    overflow: hidden;

    &:after {
      content: "";
      position: absolute;
      bottom: 0;
      right: #{$sidebar-close-width / 4};
      height: 1px;
      width: calc(100% - #{$sidebar-close-width / 2});
      background-color: hsla(0, 0%, 100%, .5);
    }

    & .sidebar-logo-link {
      padding: 0 20px;
      display: flex;
      align-items: center;
      justify-content: flex-start;
      height: 100%;
      width: 100%;

      & .logo-text {
        font-size: 32px;
        font-weight: bold;
        color: #ffffff;
        line-height: 1;
      }
    }

    &.collapse {
      .sidebar-logo-link {
        padding: 0;
      }
    }
  }
</style>
