<template>
  <layout-content :header="$t('business.tools.route_matcher.title')" :back-to="{ name: 'Dashboard' }">
    <el-card class="route-matcher-card">
      <div slot="header">
        <span>{{ $t('business.tools.route_matcher.title') }}</span>
      </div>
      
      <el-form :model="form" label-width="160px" :rules="rules" ref="form">
        <el-form-item :label="$t('business.cluster.namespace')" prop="namespace">
          <el-select 
            v-model="form.namespace" 
            :placeholder="$t('business.tools.route_matcher.select_namespace')"
            @change="onNamespaceChange"
            style="width: 100%"
          >
            <el-option
              v-for="ns in namespaces"
              :key="ns"
              :label="ns"
              :value="ns"
            />
          </el-select>
        </el-form-item>
        
        <el-form-item :label="$t('business.tools.route_matcher.test_url')" prop="testUrl">
          <el-input 
            v-model="form.testUrl" 
            :placeholder="$t('business.tools.route_matcher.url_placeholder')"
            @keyup.enter.native="matchRoute"
            style="width: 100%"
          />
        </el-form-item>
        
        <el-form-item>
          <el-button 
            type="primary"
            icon="el-icon-search" 
            @click="matchRoute"
            :loading="matching"
          >
            {{ $t('business.tools.route_matcher.match') }}
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="route-matcher-card" style="margin-top: 20px" v-if="matchResults.length > 0">
      <div slot="header">
        <span>{{ $t('business.tools.route_matcher.match_results') }}</span>
        <el-tag type="success" style="margin-left: 10px">
          {{ $t('business.tools.route_matcher.matched_count', {count: matchResults.length}) }}
        </el-tag>
      </div>
      
      <el-table :data="matchResults" border stripe>
        <el-table-column :label="$t('business.tools.route_matcher.ingress_name')" prop="ingressName" width="200" />
        <el-table-column :label="$t('business.cluster.namespace')" prop="namespace" width="150" />
        <el-table-column :label="$t('business.tools.route_matcher.host')" prop="host" width="200" />
        <el-table-column :label="$t('business.tools.route_matcher.path')" prop="path" width="200" />
        <el-table-column :label="$t('business.tools.route_matcher.path_type')" prop="pathType" width="120">
          <template slot-scope="scope">
            <el-tag :type="getPathTypeTag(scope.row.pathType)" size="small">
              {{ scope.row.pathType }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="$t('business.tools.route_matcher.service')" prop="serviceName" width="180" />
        <el-table-column :label="$t('business.tools.route_matcher.port')" prop="servicePort" width="100" />
        <el-table-column :label="$t('business.tools.route_matcher.match_type')" prop="matchType" width="120">
          <template slot-scope="scope">
            <el-tag :type="scope.row.matchType === 'exact' ? 'success' : 'info'" size="small">
              {{ scope.row.matchType === 'exact' ? $t('business.tools.route_matcher.exact_match') : $t('business.tools.route_matcher.prefix_match') }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="route-matcher-card" style="margin-top: 20px" v-if="matchResults.length === 0 && hasSearched">
      <div style="text-align: center; padding: 20px; color: #909399;">
        {{ $t('business.tools.route_matcher.no_match') }}
      </div>
    </el-card>

    <el-card class="route-matcher-card" style="margin-top: 20px" v-if="ingressList.length > 0">
      <div slot="header">
        <span>{{ $t('business.tools.route_matcher.all_rules') }}</span>
        <el-tag style="margin-left: 10px">
          {{ $t('business.tools.route_matcher.total_count', {count: ingressList.length}) }}
        </el-tag>
      </div>
      
      <el-table :data="ingressList" border stripe>
        <el-table-column :label="$t('business.tools.route_matcher.ingress_name')" prop="metadata.name" width="200" show-overflow-tooltip />
        <el-table-column :label="$t('business.cluster.namespace')" prop="metadata.namespace" width="150" />
        <el-table-column :label="$t('business.tools.route_matcher.rules')" min-width="300" show-overflow-tooltip>
          <template slot-scope="scope">
            <div v-for="(rule, ruleIndex) in scope.row.spec.rules" :key="ruleIndex" style="margin-bottom: 10px">
              <div><strong>{{ $t('business.tools.route_matcher.host') }}:</strong> {{ rule.host || '*' }}</div>
              <div v-for="(path, pathIndex) in rule.http.paths" :key="pathIndex" style="margin-left: 20px; margin-top: 5px">
                <el-tag size="mini" :type="getPathTypeTag(path.pathType)" style="margin-right: 5px">
                  {{ path.pathType }}
                </el-tag>
                <span>{{ path.path || '/' }}</span>
                <span style="margin-left: 10px; color: #909399">
                  → {{ getServiceName(path.backend) }}:{{ getServicePort(path.backend) }}
                </span>
              </div>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </layout-content>
</template>

<script>
import LayoutContent from "@/components/layout/LayoutContent"
import { listIngressWithNs } from "@/api/ingress"
import { listNamespace } from "@/api/namespaces"

export default {
  name: "RouteMatcher",
  components: { LayoutContent },
  data() {
    return {
      form: {
        namespace: "",
        testUrl: ""
      },
      rules: {},
      namespaces: [],
      ingressList: [],
      matchResults: [],
      matching: false,
      hasSearched: false,
      clusterName: ""
    }
  },
  created() {
    // 从路由或store获取集群名称
    this.clusterName = this.$route.query.cluster || this.$store.state.cluster?.clusterName || ""
    
    // 初始化表单验证规则
    this.rules = {
      namespace: [
        { required: true, message: this.$t('business.tools.route_matcher.select_namespace'), trigger: 'change' }
      ],
      testUrl: [
        { required: true, message: this.$t('business.tools.route_matcher.input_url'), trigger: 'blur' }
      ]
    }
    
    if (this.clusterName) {
      this.loadNamespaces()
    } else {
      this.$message.warning(this.$t('business.tools.route_matcher.no_cluster_selected'))
    }
  },
  methods: {
    loadNamespaces() {
      listNamespace(this.clusterName, false).then(res => {
        // 兼容不同的返回格式
        let items = []
        if (res.items) {
          items = res.items
        } else if (res.data && res.data.items) {
          items = res.data.items
        } else if (Array.isArray(res)) {
          items = res
        }
        this.namespaces = items.map(item => {
          if (item.metadata && item.metadata.name) {
            return item.metadata.name
          }
          return item
        }).filter(name => name)
      }).catch(err => {
        this.$message.error(this.$t('business.tools.route_matcher.load_namespace_failed'))
      })
    },
    onNamespaceChange() {
      this.ingressList = []
      this.matchResults = []
      this.hasSearched = false
      
      if (this.form.namespace) {
        this.loadIngresses()
      }
    },
    loadIngresses() {
      if (!this.clusterName || !this.form.namespace) {
        return
      }
      
      listIngressWithNs(this.clusterName, this.form.namespace).then(res => {
        // 兼容不同的返回格式
        if (res.items && Array.isArray(res.items)) {
          this.ingressList = res.items
        } else if (res.data && Array.isArray(res.data)) {
          this.ingressList = res.data
        } else if (Array.isArray(res)) {
          this.ingressList = res
        } else {
          this.ingressList = []
        }
      }).catch(err => {
        this.$message.error(this.$t('business.tools.route_matcher.load_ingress_failed'))
      })
    },
    matchRoute() {
      this.$refs.form.validate((valid) => {
        if (!valid) {
          return false
        }
        
        this.matching = true
        this.hasSearched = true
        this.matchResults = []
        
        try {
          const url = new URL(this.form.testUrl)
          const testHost = url.hostname
          const testPath = url.pathname
          
          const results = []
          
          for (const ingress of this.ingressList) {
            if (!ingress.spec || !ingress.spec.rules) {
              continue
            }
            
            for (const rule of ingress.spec.rules) {
              const ruleHost = rule.host || "*"
              
              // 检查host匹配
              if (ruleHost !== "*" && ruleHost !== testHost) {
                continue
              }
              
              if (!rule.http || !rule.http.paths) {
                continue
              }
              
              for (const path of rule.http.paths) {
                const pathPattern = path.path || "/"
                const pathType = path.pathType || "Prefix"
                
                let matched = false
                let matchType = ""
                
                if (pathType === "Exact") {
                  if (testPath === pathPattern) {
                    matched = true
                    matchType = "exact"
                  }
                } else if (pathType === "Prefix") {
                  if (testPath.startsWith(pathPattern)) {
                    matched = true
                    matchType = "prefix"
                  }
                } else if (pathType === "ImplementationSpecific") {
                  // ImplementationSpecific 通常也按前缀匹配处理
                  if (testPath.startsWith(pathPattern)) {
                    matched = true
                    matchType = "prefix"
                  }
                }
                
                if (matched) {
                  results.push({
                    ingressName: ingress.metadata.name,
                    namespace: ingress.metadata.namespace,
                    host: ruleHost,
                    path: pathPattern,
                    pathType: pathType,
                    serviceName: this.getServiceName(path.backend),
                    servicePort: this.getServicePort(path.backend),
                    matchType: matchType
                  })
                }
              }
            }
          }
          
          // 按匹配优先级排序：Exact优先，然后按路径长度降序
          results.sort((a, b) => {
            if (a.matchType === 'exact' && b.matchType !== 'exact') return -1
            if (a.matchType !== 'exact' && b.matchType === 'exact') return 1
            return b.path.length - a.path.length
          })
          
          this.matchResults = results
          
          if (results.length === 0) {
            this.$message.warning(this.$t('business.tools.route_matcher.no_match_found'))
          } else {
            this.$message.success(this.$t('business.tools.route_matcher.match_success', {count: results.length}))
          }
        } catch (error) {
          this.$message.error(this.$t('business.tools.route_matcher.invalid_url'))
        } finally {
          this.matching = false
        }
      })
    },
    getServiceName(backend) {
      if (!backend) return "-"
      if (backend.service && backend.service.name) {
        return backend.service.name
      }
      if (backend.serviceName) {
        return backend.serviceName
      }
      return "-"
    },
    getServicePort(backend) {
      if (!backend) return "-"
      if (backend.service && backend.service.port) {
        if (backend.service.port.number) {
          return backend.service.port.number
        }
        if (backend.service.port.name) {
          return backend.service.port.name
        }
      }
      if (backend.servicePort) {
        return backend.servicePort
      }
      return "-"
    },
    getPathTypeTag(pathType) {
      switch (pathType) {
        case "Exact":
          return "success"
        case "Prefix":
          return "primary"
        case "ImplementationSpecific":
          return "warning"
        default:
          return "info"
      }
    }
  }
}
</script>

<style lang="scss" scoped>
.route-matcher-card {
  background-color: #2a2d31;
  border: 1px solid #3e4145;
  
  ::v-deep .el-card__header {
    background-color: #1e1e1e;
    border-bottom: 1px solid #3e4145;
    color: #ffffff;
  }
  
  ::v-deep .el-card__body {
    color: #b6c0cd;
  }
  
  ::v-deep .el-form-item__label {
    color: #b6c0cd;
  }
  
  ::v-deep .el-input__inner {
    background-color: #1e1e1e;
    border-color: #3e4145;
    color: #ffffff;
  }
  
  ::v-deep .el-select .el-input__inner {
    background-color: #1e1e1e;
    border-color: #3e4145;
    color: #ffffff;
  }
  
  
}
</style>

