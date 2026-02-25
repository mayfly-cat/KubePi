FROM node:18.10.0-alpine as stage-web-build
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache make
ARG NPM_REGISTRY="https://registry.npmmirror.com"
ENV NPM_REGISTY=$NPM_REGISTRY

LABEL stage=stage-web-build
RUN set -ex \
    && npm config set registry ${NPM_REGISTRY}

WORKDIR /build/kubepi/web

COPY . .

RUN make build_web

RUN rm -fr web

FROM golang:1.22 as stage-bin-build

ENV GOPROXY="https://goproxy.cn,direct"

ENV CGO_ENABLED=0

ENV GO111MODULE=on

LABEL stage=stage-bin-build

WORKDIR /build/kubepi/bin

COPY --from=stage-web-build /build/kubepi/web .

RUN go mod download

RUN make build_gotty
RUN make build_bin

FROM alpine:3.16

WORKDIR /

COPY --from=stage-bin-build /build/kubepi/bin/dist/usr /usr
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 设置架构变量并安装基础工具
RUN ARCH=$(uname -m) \
    && case $ARCH in \
        aarch64) ARCH="arm64";; \
        x86_64) ARCH="amd64";; \
        *) echo "Unsupported architecture: $ARCH"; exit 1;; \
    esac \
    && echo "Detected ARCH: $ARCH" \
    && apk add --update --no-cache bash bash-completion curl wget openssl iputils busybox-extras vim tini ca-certificates \
    && update-ca-certificates \
    && sed -i "s/nobody:\//nobody:\/nonexistent/g" /etc/passwd

# 下载 kubectl（使用独立的 RUN 命令以避免复杂的嵌套逻辑）
RUN ARCH=$(uname -m) \
    && case $ARCH in \
        aarch64) ARCH="arm64";; \
        x86_64) ARCH="amd64";; \
        *) ARCH="amd64";; \
    esac \
    && KUBECTL_VERSION="v1.22.1" \
    && PRIMARY_URL="https://kubeoperator.oss-cn-beijing.aliyuncs.com/kubepi/kubectl/${KUBECTL_VERSION}/${ARCH}/kubectl" \
    && FALLBACK_URL="https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl" \
    && echo "Attempting to download kubectl (ARCH: ${ARCH}, VERSION: ${KUBECTL_VERSION})" \
    && DOWNLOADED=false \
    && echo "Trying primary URL with curl..." \
    && (curl -sLf --connect-timeout 15 --max-time 90 "$PRIMARY_URL" -o /usr/bin/kubectl \
        || (sleep 3 && curl -sLf --connect-timeout 15 --max-time 90 "$PRIMARY_URL" -o /usr/bin/kubectl) \
        || (sleep 3 && curl -sLf --connect-timeout 15 --max-time 90 "$PRIMARY_URL" -o /usr/bin/kubectl)) \
    && if [ -f /usr/bin/kubectl ] && [ -s /usr/bin/kubectl ]; then \
        DOWNLOADED=true; \
        echo "kubectl downloaded successfully from primary URL (curl)"; \
    fi \
    && if [ "$DOWNLOADED" != "true" ]; then \
        echo "curl failed, trying wget for primary URL (with --no-check-certificate)..." \
        && wget --no-check-certificate --timeout=30 --tries=2 "$PRIMARY_URL" -O /usr/bin/kubectl 2>&1 \
        && if [ -f /usr/bin/kubectl ] && [ -s /usr/bin/kubectl ]; then \
            DOWNLOADED=true; \
            echo "kubectl downloaded successfully from primary URL (wget)"; \
        fi; \
    fi \
    && if [ "$DOWNLOADED" != "true" ]; then \
        echo "Primary URL failed, trying fallback URL: $FALLBACK_URL" \
        && (curl -sLf --connect-timeout 15 --max-time 90 "$FALLBACK_URL" -o /usr/bin/kubectl \
            || (sleep 3 && curl -sLf --connect-timeout 15 --max-time 90 "$FALLBACK_URL" -o /usr/bin/kubectl) \
            || (sleep 3 && curl -sLf --connect-timeout 15 --max-time 90 "$FALLBACK_URL" -o /usr/bin/kubectl)) \
        && if [ -f /usr/bin/kubectl ] && [ -s /usr/bin/kubectl ]; then \
            DOWNLOADED=true; \
            echo "kubectl downloaded successfully from fallback URL"; \
        fi; \
    fi \
    && if [ "$DOWNLOADED" != "true" ]; then \
        echo "ERROR: Failed to download kubectl from all sources"; \
        echo "Primary URL: $PRIMARY_URL"; \
        echo "Fallback URL: $FALLBACK_URL"; \
        exit 1; \
    fi \
    && chmod +x /usr/bin/kubectl \
    && echo "Verifying kubectl installation..." \
    && /usr/bin/kubectl version --client --short \
    && echo "kubectl installed successfully"

# 安装 kubectl-aliases
RUN cd /opt/ \
    && wget -q --show-progress --progress=bar:force https://kubeoperator.oss-cn-beijing.aliyuncs.com/kubepi/kubectl-aliases/kubectl-aliases.tar.gz \
    && tar zxvf kubectl-aliases.tar.gz \
    && rm -rf kubectl-aliases.tar.gz \
    && chmod -R 755 kubectl-aliases \
    || (echo "Failed to install kubectl-aliases" && exit 1)

# 安装 fzf (可选，如果安装失败则跳过)
RUN ARCH=$(uname -m) \
    && case $ARCH in \
        aarch64) ARCH="arm64";; \
        x86_64) ARCH="amd64";; \
        *) ARCH="amd64";; \
    esac \
    && cd /opt/ \
    && if wget -q --show-progress --progress=bar:force https://kubeoperator.oss-cn-beijing.aliyuncs.com/kubepi/fzf/0.21.0/fzf.tar.gz; then \
        tar zxvf fzf.tar.gz \
        && rm -rf fzf.tar.gz \
        && echo "Listing /opt/ contents after extraction:" \
        && ls -la /opt/ \
        && echo "Listing fzf directory contents:" \
        && (ls -la fzf/ 2>/dev/null || echo "fzf directory not found") \
        && chmod -R 755 fzf 2>/dev/null || true \
        && if [ -f fzf/install ]; then \
            echo "Running fzf install script..." \
            && (cd fzf && ./install --bin 2>&1 || echo "fzf install script completed with warnings"); \
        else \
            echo "fzf install script not found, searching for binary..."; \
        fi \
        && FZF_BINARY="" \
        && if [ -f fzf/bin/fzf ]; then \
            FZF_BINARY="fzf/bin/fzf"; \
        elif [ -f fzf/fzf ]; then \
            FZF_BINARY="fzf/fzf"; \
        else \
            FZF_FOUND=$(find fzf -name "fzf" -type f 2>/dev/null | head -1) \
            && if [ -n "$FZF_FOUND" ]; then \
                FZF_BINARY="$FZF_FOUND"; \
            fi; \
        fi \
        && if [ -n "$FZF_BINARY" ] && [ -f "$FZF_BINARY" ]; then \
            echo "Found fzf binary at: $FZF_BINARY" \
            && chmod +x "$FZF_BINARY" \
            && ln -sf "/opt/$FZF_BINARY" /usr/local/bin/fzf \
            && echo "fzf installed successfully"; \
        else \
            echo "WARNING: fzf binary not found. Directory structure:" \
            && find /opt -name "*fzf*" 2>/dev/null || echo "No fzf files found" \
            && echo "fzf installation skipped, continuing build..."; \
        fi; \
    else \
        echo "WARNING: Failed to download fzf.tar.gz, skipping fzf installation..."; \
    fi

# 安装 k9s
RUN ARCH=$(uname -m) \
    && case $ARCH in aarch64) ARCH="arm64";; x86_64) ARCH="amd64";; esac \
    && cd /tmp/ \
    && wget -q --show-progress --progress=bar:force https://kubeoperator.oss-cn-beijing.aliyuncs.com/kubepi/k9s/v0.24.14/k9s_Linux_${ARCH}.tar.gz \
    && tar -xvf k9s_Linux_${ARCH}.tar.gz \
    && chmod +x k9s \
    && mv k9s /usr/bin \
    && echo "k9s installed successfully"

# 安装 kubens
RUN ARCH=$(uname -m) \
    && case $ARCH in aarch64) ARCH="arm64";; x86_64) ARCH="amd64";; esac \
    && KUBECTX_VERSION=v0.9.4 \
    && cd /tmp/ \
    && wget -q --show-progress --progress=bar:force https://kubeoperator.oss-cn-beijing.aliyuncs.com/kubepi/kubens/${KUBECTX_VERSION}/kubens_${KUBECTX_VERSION}_linux_${ARCH}.tar.gz \
    && tar -xvf kubens_${KUBECTX_VERSION}_linux_${ARCH}.tar.gz \
    && chmod +x kubens \
    && mv kubens /usr/bin \
    && echo "kubens installed successfully"

# 安装 kubectx
RUN ARCH=$(uname -m) \
    && case $ARCH in aarch64) ARCH="arm64";; x86_64) ARCH="amd64";; esac \
    && KUBECTX_VERSION=v0.9.4 \
    && cd /tmp/ \
    && wget -q --show-progress --progress=bar:force https://kubeoperator.oss-cn-beijing.aliyuncs.com/kubepi/kubectx/${KUBECTX_VERSION}/kubectx_${KUBECTX_VERSION}_linux_${ARCH}.tar.gz \
    && tar -xvf kubectx_${KUBECTX_VERSION}_linux_${ARCH}.tar.gz \
    && chmod +x kubectx \
    && mv kubectx /usr/bin \
    && echo "kubectx installed successfully"

# 安装 helm
RUN ARCH=$(uname -m) \
    && case $ARCH in aarch64) ARCH="arm64";; x86_64) ARCH="amd64";; esac \
    && HELM_VERSION=v3.10.2 \
    && cd /tmp/ \
    && wget -q --show-progress --progress=bar:force http://kubeoperator.oss-cn-beijing.aliyuncs.com/helm/${HELM_VERSION}/helm-${HELM_VERSION}-linux-${ARCH}.tar.gz \
    && tar -xvf helm-${HELM_VERSION}-linux-${ARCH}.tar.gz \
    && mv linux-${ARCH}/helm /usr/local/bin \
    && chmod +x /usr/local/bin/helm \
    && echo "helm installed successfully"

# 设置权限和清理
RUN chmod +x /usr/local/bin/gotty \
    && chmod 555 /bin/busybox \
    && rm -rf /tmp/* /var/tmp/* /var/cache/apk/* \
    && chmod -R 755 /tmp \
    && mkdir -p /opt/webkubectl

RUN apk add tzdata && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

COPY conf/app.yml /etc/kubepi/app.yml

COPY vimrc.local /etc/vim

EXPOSE 80

USER root

ENTRYPOINT ["tini", "-g", "--"]
CMD ["kubepi-server","-c", "/etc/kubepi" ,"--server-bind-host","0.0.0.0","--server-bind-port","80"]