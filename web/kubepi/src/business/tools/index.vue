<template>
  <layout-content :header="$t('business.tools.title')">
    <el-tabs v-model="activeTab" type="card">
      <el-tab-pane :label="$t('business.tools.quick_access')" name="quick">
        <div class="tools-container">
          <el-row :gutter="20">
            <el-col :xs="24" :sm="12" :md="8" :lg="8" v-for="tool in tools" :key="tool.name">
              <el-card 
                class="tool-card" 
                shadow="hover"
                @click.native="openTool(tool.url)"
              >
                <div class="tool-card-content">
                  <div class="tool-icon">
                    <i :class="tool.icon" class="icon-large"></i>
                  </div>
                  <div class="tool-info">
                    <h3 class="tool-title">{{ tool.title }}</h3>
                    <p class="tool-desc">{{ tool.description }}</p>
                  </div>
                </div>
              </el-card>
            </el-col>
          </el-row>
        </div>
      </el-tab-pane>
    </el-tabs>
  </layout-content>
</template>

<script>
import LayoutContent from "@/components/layout/LayoutContent";

export default {
  name: "ToolsIndex",
  components: { LayoutContent },
  data() {
    return {
      activeTab: "quick"
    };
  },
  computed: {
    tools() {
      return [
        {
          name: "grafana",
          title: this.$t("business.tools.grafana.title"),
          description: this.$t("business.tools.grafana.description"),
          url: "https://grafana.gateway-sop.gaia.carizon.com",
          icon: "el-icon-data-analysis"
        },
        {
          name: "logs",
          title: this.$t("business.tools.logs.title"),
          description: this.$t("business.tools.logs.description"),
          url: "https://visual-o-008eem0wa8h7.escloud.volces.com:5601",
          icon: "el-icon-document"
        },
        {
          name: "tracing",
          title: this.$t("business.tools.tracing.title"),
          description: this.$t("business.tools.tracing.description"),
          url: "https://tracing.gateway-sop.gaia.carizon.com",
          icon: "el-icon-connection"
        }
      ];
    }
  },
  methods: {
    openTool(url) {
      window.open(url, "_blank");
    }
  }
};
</script>

<style lang="scss" scoped>
.tools-container {
  padding: 20px 0;
}

.tool-card {
  margin-bottom: 20px;
  cursor: pointer;
  transition: all 0.3s;
  border: 1px solid #3e4145;
  background-color: #2a2d31;

  &:hover {
    transform: translateY(-5px);
    box-shadow: 0 4px 12px rgba(56, 132, 197, 0.3);
    border-color: #3884c5;
  }
}

.tool-card-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 20px;
  min-height: 200px;
}

.tool-icon {
  margin-bottom: 20px;

  .icon-large {
    font-size: 48px;
    color: #3884c5;
  }
}

.tool-info {
  flex: 1;
  width: 100%;
}

.tool-title {
  font-size: 18px;
  font-weight: 600;
  color: #ffffff;
  margin: 0 0 10px 0;
}

.tool-desc {
  font-size: 14px;
  color: #b6c0cd;
  margin: 0;
  line-height: 1.6;
}

::v-deep .el-tabs__header {
  margin-bottom: 20px;
}

::v-deep .el-tabs__item {
  color: #b6c0cd;
  
  &.is-active {
    color: #3884c5;
  }
}

::v-deep .el-card__body {
  padding: 0;
}
</style>

