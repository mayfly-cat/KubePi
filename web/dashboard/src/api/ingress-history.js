import {get, post} from "@/plugins/request"

// 获取 Ingress 历史版本列表
export function listIngressHistory(clusterName, namespace, ingressName) {
  return get(`/api/v1/clusters/${clusterName}/namespaces/${namespace}/ingresses/${ingressName}/history`)
}

// 获取指定版本的历史记录
export function getIngressHistory(clusterName, namespace, ingressName, version) {
  return get(`/api/v1/clusters/${clusterName}/namespaces/${namespace}/ingresses/${ingressName}/history/${version}`)
}

// 回滚到指定版本
export function rollbackIngress(clusterName, namespace, ingressName, version) {
  return post(`/api/v1/clusters/${clusterName}/namespaces/${namespace}/ingresses/${ingressName}/history/${version}/rollback`)
}

