<template>
  <el-container class="home-container">
    <el-aside :width="isCollapse ? '64px' : '200px'">
      <div class="logo">
        <img src="../assets/image/DevOps平台.svg" class="siderbar-logo">
        <h2 v-show="!isCollapse">AutoOps</h2>
      </div>
      <el-menu background-color="transparent" text-color="rgba(255,255,255,0.75)" active-text-color="#ffffff" router :default-active="$route.path"
               :collapse="isCollapse" :collapse-transition="false" class="sidebar-menu">
        <!--无子集菜单-->
        <el-menu-item :index="'/' + item.url" v-for="item in noChildren" :key="item.menuName" @click="saveNavState('/' + item.url)">
          <el-icon><component :is="item.icon" /></el-icon>
          <template v-slot:title>
            <span>{{ item.menuName }}</span>
          </template>
        </el-menu-item>
        <!--有子集菜单-->
        <el-sub-menu :index="item.id + ''" v-for="item in hasChildren" :key="item.id">
          <template #title>
            <el-icon><component :is="item.icon" /></el-icon>
            <span>{{ item.menuName }}</span>
          </template>
          <el-menu-item :index="'/' + subItem.url" v-for="subItem in item.menuSvoList" :key="subItem.id"
                        @click="saveNavState('/' + subItem.url)">
            <el-icon><component :is="subItem.icon" /></el-icon>
            <template #title>
              <span>{{ subItem.menuName }}</span>
            </template>
          </el-menu-item>
        </el-sub-menu>

      </el-menu>
    </el-aside>

    <!-- 主体内容 -->
    <el-container>
      <el-header height="50px">
        <div class="header-left">
          <el-button text @click="toggleCollapse" class="collapse-btn">
            <el-icon size="20"><component :is="collapseBtnClass" /></el-icon>
          </el-button>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">仪表盘</el-breadcrumb-item>
            <el-breadcrumb-item v-if="$route.meta && $route.meta.sTitle">
              {{ $route.meta.sTitle }}
            </el-breadcrumb-item>
            <el-breadcrumb-item v-if="$route.meta && $route.meta.tTitle">
              {{ $route.meta.tTitle }}
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <HeadImage />
      </el-header>
      <Tags />
      <el-main><router-view /></el-main>
    </el-container>
  </el-container>
</template>

<script>


import storage from "@/utils/storage";
import HeadImage   from "@/components/HeadImage.vue";
import Tags from "@/components/Tags.vue";

export default {
  // eslint-disable-next-line vue/multi-word-component-names
  name: "Home",
  components: { HeadImage, Tags },
  data() {
    return {
      leftMenuList: null,
      activePath: '',
      collapseBtnClass: "Fold",
      isCollapse: false,
    }
  },
  computed: {
    noChildren() {
      return (this.leftMenuList || []).filter(item => !item.menuSvoList)
    },
    hasChildren() {
      return (this.leftMenuList || []).filter(item => item.menuSvoList)
    }
  },
  methods: {
    initMenuData() {
      try {
        const menuData = storage.getItem("leftMenuList");

        if (Array.isArray(menuData)) {
          this.leftMenuList = menuData;
          
          const taskMenu = this.leftMenuList.find(item => item.menuName === '任务中心')
          if (taskMenu && taskMenu.menuSvoList) {
            const configExists = taskMenu.menuSvoList.some(sub => sub.url === 'task/config')
            if (!configExists) {
              taskMenu.menuSvoList.push({
                id: 99999,
                menuName: '配置管理',
                url: 'task/config',
                icon: 'Setting'
              })
            }
          }
        } else if (menuData) {
          this.leftMenuList = [];
        } else {
          this.leftMenuList = [];
        }

        this.$forceUpdate();
      } catch (error) {
        console.error('初始化菜单数据失败:', error);
        this.leftMenuList = [];
      }
    },
    saveNavState(activePath) {
      storage.setItem('activePath', activePath)
      this.activePath = activePath
    },
    toggleCollapse() {
      this.isCollapse = !this.isCollapse
      if (this.isCollapse) {
        this.collapseBtnClass = 'Fold'
      } else {
        this.collapseBtnClass = 'Expand'
      }
    }
  },
  mounted() {
    this.initMenuData();
  }
}
</script>

<style lang="less" scoped>
.home-container {
  height: 100%;

  // ═══════════════════════════════════════════════════════════
  //  侧边栏 — 沉稳深蓝灰，无渐变，无玻璃拟态
  // ═══════════════════════════════════════════════════════════
  .el-aside {
    background-color: var(--ao-sidebar-bg);
    transition: width var(--ao-transition);
    overflow-x: hidden;
    overflow-y: auto;

    .logo {
      display: flex;
      align-items: center;
      height: var(--ao-header-height);
      padding: 0 16px;
      white-space: nowrap;
      overflow: hidden;

      .siderbar-logo {
        width: 32px;
        height: 32px;
        flex-shrink: 0;
      }

      h2 {
        margin: 0 0 0 10px;
        font-size: 16px;
        font-weight: 600;
        color: rgba(255, 255, 255, 0.9);
        letter-spacing: 0.5px;
        flex-shrink: 0;
      }
    }

    // 菜单全局
    .sidebar-menu {
      border-right: none;
    }
  }

  // ═══════════════════════════════════════════════════════════
  //  菜单项样式 — 克制、无位移动效
  // ═══════════════════════════════════════════════════════════
  .sidebar-menu {
    .el-menu-item {
      height: 44px;
      line-height: 44px;
      margin: 2px 8px;
      border-radius: var(--ao-radius);
      color: var(--ao-sidebar-text);
      transition: background-color var(--ao-transition), color var(--ao-transition);

      &:hover {
        background-color: rgba(255, 255, 255, 0.08) !important;
        color: var(--ao-sidebar-text-hover);
      }

      &:focus, &:focus-visible {
        outline: none;
      }
    }

    // 一级菜单激活态
    > .el-menu-item.is-active {
      background-color: var(--ao-sidebar-active-bg) !important;
      color: var(--ao-sidebar-active-text) !important;
      border-left: 3px solid var(--ao-sidebar-active-border);
      border-radius: 0 var(--ao-radius) var(--ao-radius) 0;
      margin-left: 0;
      padding-left: 17px; // 20px - 3px border
    }

    // 子菜单标题
    .el-sub-menu {
      .el-sub-menu__title {
        height: 44px;
        line-height: 44px;
        margin: 2px 8px;
        border-radius: var(--ao-radius);
        color: var(--ao-sidebar-text);
        transition: background-color var(--ao-transition), color var(--ao-transition);

        &:hover {
          background-color: rgba(255, 255, 255, 0.08) !important;
          color: var(--ao-sidebar-text-hover);
        }

        &:focus, &:focus-visible {
          outline: none;
        }
      }

      // 子菜单展开背景
      &.is-opened > .el-sub-menu__title {
        color: var(--ao-sidebar-text-hover);
      }

      // 二级菜单项
      .el-menu-item {
        height: 40px;
        line-height: 40px;
        margin: 1px 8px 1px 12px;
        font-size: 13px;

        &.is-active {
          background-color: var(--ao-sidebar-active-bg) !important;
          color: var(--ao-sidebar-active-text) !important;
          border-left: 3px solid var(--ao-sidebar-active-border);
          border-radius: 0 var(--ao-radius) var(--ao-radius) 0;
          margin-left: 4px;
          padding-left: 17px;
        }
      }
    }

    // 父菜单展开/激活不需要特殊背景
    > .el-sub-menu.is-active,
    > .el-sub-menu.is-opened {
      background-color: transparent !important;
    }
  }

  // ═══════════════════════════════════════════════════════════
  //  顶栏 — 纯白底 + 底部分割线
  // ═══════════════════════════════════════════════════════════
  .el-header {
    background-color: var(--ao-header-bg);
    border-bottom: 1px solid var(--ao-header-border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 var(--ao-page-padding);

    .header-left {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .collapse-btn {
      color: var(--ao-text-secondary);
      padding: 6px;

      &:hover {
        color: var(--ao-primary);
      }
    }
  }

  // ═══════════════════════════════════════════════════════════
  //  主内容区
  // ═══════════════════════════════════════════════════════════
  .el-main {
    background-color: var(--ao-bg-page);
  }
}
</style>

<style lang="less">
// 折叠菜单弹出层 — 与侧边栏保持一致的深蓝灰
.el-menu--popup-bottom-start,
.el-menu--popup {
  background-color: var(--ao-sidebar-bg) !important;
  border: 1px solid rgba(255, 255, 255, 0.08) !important;
  box-shadow: var(--ao-shadow-lg) !important;
  border-radius: var(--ao-radius) !important;

  .el-menu-item {
    color: var(--ao-sidebar-text) !important;
    background: transparent !important;
    transition: background-color var(--ao-transition);
    margin: 2px 8px !important;
    border-radius: var(--ao-radius) !important;

    &:hover {
      background-color: rgba(255, 255, 255, 0.08) !important;
      color: var(--ao-sidebar-text-hover) !important;
    }

    &.is-active {
      background-color: var(--ao-sidebar-active-bg) !important;
      color: var(--ao-sidebar-active-text) !important;
    }

    .el-icon,
    span {
      color: inherit !important;
    }
  }
}
</style>
