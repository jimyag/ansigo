# Dockerfile for target SSH node
FROM ubuntu:24.04

# 安装 SSH 服务器和 Python（Ansible 模块依赖）
RUN apt-get update && \
    apt-get install -y \
    openssh-server \
    python3 \
    python3-pip \
    sudo \
    curl \
    && apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# 配置 SSH
RUN mkdir /var/run/sshd && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config

# 创建测试用户
RUN useradd -m -s /bin/bash testuser && \
    echo 'testuser:testpass' | chpasswd && \
    usermod -aG sudo testuser && \
    echo 'testuser ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

# 配置 SSH 密钥目录
RUN mkdir -p /home/testuser/.ssh && \
    chmod 700 /home/testuser/.ssh && \
    chown testuser:testuser /home/testuser/.ssh

EXPOSE 22

CMD ["/usr/sbin/sshd", "-D"]
