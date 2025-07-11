根据您提供的 e2b.Dockerfile，执行 e2b template build 后的产物是一个自定义的 E2B 沙箱环境镜像。


  这个镜像包含了 e2bdev/code-interpreter:latest 基础镜像的所有内容，并且额外通过 pip 安装了 cowsay 库。


  这个产物（镜像）会被存储起来，之后您可以基于这个定制的环境快速启动新的沙箱实例。

## 沙箱实例具体是怎么一步一步的启动的

✦ 好的，E2B 沙箱实例的启动过程可以分为以下几个核心步骤，这个过程连接了您的本地代码和云端的安全执行环境：

  启动流程概览



   1 您本地的代码          <-- (通过SDK) -->          E2B 云平台          <-- (管理) -->
     一个隔离的沙箱实例
   2 (Agent)
     (基于您的Dockerfile构建)


  ---

  详细分步说明：

  第 1 步：在您的代码中请求沙箱


  您不会手动去某个地方点击“启动”按钮。启动过程完全由代码驱动，通过 E2B 提供的 SDK (例如 e2b for Python 或
  @e2b/sdk for TypeScript/JavaScript) 来完成。

  在您的代码里，您会写下类似这样的指令：

  TypeScript 示例 (`@e2b/sdk`)


   1 import { Sandbox } from '@e2b/sdk';
   2 
   3 // 1. 这行代码是启动的起点
   4 const sandbox = await Sandbox.create({ template: 'your-template-id' });
   5 // 'your-template-id' 就是您在 e2b.toml 中定义的 id
   6 
   7 // ... 接下来就可以与沙箱交互了 ...
   8 
   9 await sandbox.close();


  这行 Sandbox.create() 就是整个流程的触发器。

  第 2 步：SDK 向 E2B 云发送请求


  当您的代码执行到 Sandbox.create() 时，E2B 的 SDK 会做以下事情：
   1. 读取您的 API 密钥进行身份验证。
   2. 向 E2B 的云端 API 发送一个安全的 HTTPS 请求，请求启动一个基于 your-template-id 的沙箱实例。


  第 3 步：E2B 云平台进行资源调配


  E2B 的云后端接收到您的请求后，会立即执行以下操作：
   1. 验证请求：检查您的 API 密钥和请求的模板 ID 是否有效。
   2. 查找镜像：根据模板 ID，找到您之前通过 e2b template build 构建并上传的那个沙箱环境镜像 (包含了 cowsay
      的那个)。
   3. 分配资源：在一个隔离、安全的环境中，快速启动一台微型虚拟机
      (MicroVM)。这确保了每个用户的每个会话都是完全独立的。
   4. 运行容器：在这台微型虚拟机内部，从您的镜像启动一个容器。此时，您 e2b.Dockerfile
      中定义的环境（包括基础系统和 cowsay）就完全准备好了。

  第 4 步：建立安全连接


  一旦云端的沙箱环境准备就绪（这通常只需要一两秒），E2B 平台会：
   1. 在您的本地代码和云端沙箱之间建立一个持久化、双向的 WebSocket 连接。
   2. Sandbox.create() 这个异步函数调用此时才算完成，它会返回一个 sandbox 对象。

  这个 sandbox 对象就是您本地代码与远程云端沙箱进行交互的“遥控器”。

  第 5 步：与沙箱进行交互


  现在，您可以通过 sandbox 对象来“遥控”云端的沙箱了。
  例如：
   * sandbox.process.start('cowsay "Hello from E2B"')：这个命令会通过 WebSocket 发送到云端沙箱，在其中执行
     cowsay 命令，然后把结果（stdout/stderr）再通过 WebSocket 发回给您的本地代码。
   * sandbox.filesystem.write('hello.txt', 'Hello World')：这会在云端沙箱的文件系统里创建一个 hello.txt 文件。

  第 6 步：关闭与销毁


  当您完成了所有操作，可以调用 sandbox.close()。
   1. 这个调用会通知 E2B 云平台您已经使用完毕。
   2. WebSocket 连接被断开。
   3. E2B 平台会立即销毁为该会话创建的整个微型虚拟机和容器。


  关键点：沙箱是无状态的 (Stateless)。每次 Sandbox.create() 都会得到一个全新的、干净的、基于您 Dockerfile
  的环境。一旦 close()，所有在沙箱中做的文件修改、进程运行等都会被彻底清除，不会保留到下一次会话。

## e2b云端启动的细节流程是什么样的

✦ 好的，这是一个非常棒的问题。深入了解 E2B 云端的内部启动流程，可以帮助您更好地理解其速度、安全性和工作原理。


  这个流程可以类比为一个高度自动化的“云端厨房”，您（通过 SDK）下单，厨房（E2B
  云平台）立即为您准备一个专属、干净的烹饪台（沙箱）。

  以下是 E2B 云端启动一个沙箱实例的详细内部流程，涉及的关键技术包括 微型虚拟机 (MicroVMs) 和 容器化。

  核心架构：控制平面 vs. 执行平面


  首先，需要理解 E2B 云平台有两个主要部分：
   1. 控制平面 (Control Plane)：这是“大脑”和“调度中心”。它负责接收 API 请求、验证用户、管理资源和下达指令。
   2. 执行平面 (Execution Plane)：这是一组物理服务器（称为“宿主机”），它们是实际运行沙箱的“车间”。

  ---

  详细启动流程

  第 0 步：【准备工作】镜像的构建与存储


  在您运行 e2b template build 时，E2B 已经完成了以下工作：
   * 您的 e2b.Dockerfile 被发送到 E2B 的构建服务。
   * 云端根据您的 Dockerfile 构建一个标准的 OCI (Open Container Initiative) 容器镜像。
   * 这个构建好的镜像（包含了 cowsay）被存储在 E2B 的私有容器镜像仓库中，并与您的账户和模板 ID 关联起来。

  这个镜像就是未来所有沙箱实例的“模具”。

  ---

  第 1 步：API 请求到达控制平面


  当您的代码执行 await Sandbox.create({ template: '...' }) 时，一个 HTTPS 请求被发送到 E2B 的 API 网关。

  第 2 步：控制平面进行验证和调度


  控制平面接收到请求后，立即执行：
   1. 认证 (Authentication)：检查您的 API 密钥，确认您的身份。
   2. 授权 (Authorization)：检查您的账户状态、套餐配额（例如，是否达到最大并发沙箱数）。
   3. 查找镜像：在内部数据库中查找您请求的模板 ID，找到它在私有镜像仓库中的确切地址（例如
      registry.e2b.internal/user-123/template-abc:latest）。
   4. 选择宿主机：控制平面会从执行平面的服务器集群中，根据负载、地理位置等因素，选择一台最合适的、有可用资源的
      物理宿主机来创建您的沙箱。


  第 3 步：在宿主机上启动微型虚拟机 (MicroVM)

  这是实现速度和安全隔离的关键步骤。E2B 不会直接在宿主机上运行您的容器，而是先启动一个极度轻量化的虚拟机。


   1. 下达指令：控制平面向选定的宿主机发送指令：“为用户 XXX 启动一个沙箱”。
   2. 启动 Firecracker：宿主机使用像 AWS Firecracker 这样的虚拟化技术，启动一个 MicroVM。
       * 为什么是 MicroVM？ 它被设计用于无服务器计算，启动时间在毫秒级别，远快于传统的虚拟机（如 VirtualBox 或
          VMware）。
       * 安全隔离：每个 MicroVM 都有自己独立的内核、内存和虚拟设备，提供了与宿主机以及其他用户沙箱之间硬件级别
         的强隔离。这确保了您的代码绝对无法影响到其他用户。

  第 4 步：在 MicroVM 内部启动您的容器


  现在，我们有了一个空的、正在运行的、被完全隔离的 MicroVM。
   1. 拉取镜像：宿主机上的代理进程，会从 E2B 的私有镜像仓库中，将您在第 0 步构建的那个容器镜像拉取到 MicroVM
      的文件系统里。
   2. 启动容器：在 MicroVM 内部，使用 containerd 这样的标准容器运行时，从您的镜像启动一个容器。
       * 此时，您 Dockerfile 中定义的所有环境（Debian 基础系统、code-interpreter 工具、cowsay
         库）都已完全就绪并运行起来。

  第 5 步：建立 WebSocket 通信桥梁


  您的代码需要一种方式与这个远在云端、层层隔离的沙箱进行实时通信。
   1. 内部代理启动：在您的沙箱容器启动时，一个 E2B 的轻量级代理服务也会随之启动。
   2. 建立出站连接：这个内部代理会主动向 E2B 控制平面的WebSocket 网关发起一个安全的 WebSocket 连接。
   3. 连接配对：您的本地 SDK 也已经与这个 WebSocket 网关建立了连接。网关此时会将这两个连接“桥接”起来，形成一个
      从您的本地代码到云端沙箱内部的安全双向通信隧道。

  第 6 步：返回“就绪”信号


  当整个流程（MicroVM 启动 -> 容器运行 -> WebSocket 隧道建立）全部完成后，控制平面会通过 API 响应，告诉您的
  SDK：“沙箱已准备就绪”。

  此时，您代码中的 await Sandbox.create() 调用才算执行完毕，并返回一个可用的 sandbox 对象。

  总结：为什么是这样设计？


   * 速度：MicroVM 的毫秒级启动 + 预构建的容器镜像 = 极快的冷启动速度（通常 1-2 秒）。
   * 安全：MicroVM + 容器 的双重隔离机制提供了业界顶级的安全保障。您的代码在一个“盒子里的盒子”里运行。
   * 可扩展性：控制平面和执行平面的分离架构，使得 E2B 可以轻松地增减物理服务器来应对海量用户的并发请求。
   * 易用性：所有这些复杂的底层操作都对最终用户透明，您只需要关心您的 Dockerfile 和 SDK 调用即可。

## > how to directly use firecracker and docker image to start microVM

✦ Of course. This is an excellent, hands-on question that gets to the heart of virtualization and
  containerization.

  Directly using a Docker image to boot a Firecracker MicroVM is not a one-step process because a Docker image
   is not a bootable disk image.


   * A Docker Image is a layered archive of a filesystem and some metadata (like the default command to run).
     It does not contain a Linux kernel or a bootloader.
   * A Firecracker MicroVM needs two essential things to start:
       1. A Linux Kernel binary to boot.
       2. A Root Filesystem (rootfs), typically in the form of a block device (like an .ext4 disk image), that
          the kernel can mount and use.

  The core task is to convert the Docker image's filesystem into a rootfs file that Firecracker can use.


  Here is a step-by-step guide on how to do this manually on a Linux machine with KVM enabled.

  Prerequisites


  You need a Linux host with the following installed:
   1. KVM: Kernel-based Virtual Machine support. (Check with kvm-ok)
   2. Docker: To pull and manage the Docker image.
   3. Firecracker: The firecracker binary. You can download it from their GitHub releases 
      (https://github.com/firecracker-microvm/firecracker/releases).
   4. Common utilities: curl, dd, mkfs.ext4, mount.

  ---

  The Process: Step-by-Step

  Let's use the alpine:latest Docker image as our example.

  Step 1: Pull the Docker Image

  First, ensure you have the image locally.



   1 docker pull alpine:latest


  Step 2: Extract the Docker Image's Filesystem

  We need to get all the files out of the Docker image. The easiest way is to create a container from the
  image and then export its filesystem as a .tar archive.



   1 # Create a container instance (it doesn't need to be running)
   2 CONTAINER_ID=$(docker create alpine:latest)
   3 
   4 # Export the container's filesystem to a tarball
   5 docker export $CONTAINER_ID > alpine.tar
   6 
   7 # Clean up the container
   8 docker rm $CONTAINER_ID

  Now you have alpine.tar, which is a flattened archive of the entire Alpine Linux filesystem.

  Step 3: Create a Root Filesystem Image (rootfs.ext4)


  Firecracker needs a block device. We'll create a file on our host, format it as ext4, and copy the Alpine
  files into it.



    1 # 1. Create an empty file to act as our disk (e.g., 512MB)
    2 dd if=/dev/zero of=rootfs.ext4 bs=1M count=512
    3 
    4 # 2. Format this file as an ext4 filesystem
    5 mkfs.ext4 rootfs.ext4
    6 
    7 # 3. Create a temporary mount point
    8 mkdir -p /tmp/my-rootfs
    9 
   10 # 4. Mount the filesystem image to the mount point
   11 sudo mount rootfs.ext4 /tmp/my-rootfs
   12 
   13 # 5. Unpack the Docker filesystem into the mounted image
   14 sudo tar -xf alpine.tar -C /tmp/my-rootfs
   15 
   16 # 6. Unmount the image - it's now ready!
   17 sudo umount /tmp/my-rootfs

  You now have a file named rootfs.ext4 that contains the complete Alpine filesystem, ready to be used by a
  VM.

  Step 4: Get a Linux Kernel


  Firecracker needs a kernel to boot. You can't use the one from your host directly. You need a separate,
  uncompressed kernel binary (vmlinux). You can download a pre-built one or compile it yourself.

  For simplicity, let's download a generic one. The Firecracker team provides some for testing.



   1 # Download a compatible kernel (e.g., v5.10)
   2 wget https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.6/x86_64/vmlinux-5.10.186 -O
     vmlinux.bin


  Step 5: Configure and Launch the Firecracker MicroVM


  Firecracker is controlled via an API over a UNIX socket. We'll start Firecracker, then use curl to send it
   configuration commands.

  1. Start the Firecracker process:
  Open a new terminal and run:


   1 # This creates the API socket and waits for commands
   2 ./firecracker --api-sock /tmp/firecracker.socket


  2. Configure and boot the VM:
  In your original terminal, send the API commands.



    1 # API socket path
    2 API_SOCK="/tmp/firecracker.socket"
    3 
    4 # Set the kernel
    5 curl --unix-socket $API_SOCK -i \
    6     -X PUT 'http://localhost/boot-source' \
    7     -H 'Accept: application/json' \
    8     -H 'Content-Type: application/json' \
    9     -d '{
   10         "kernel_image_path": "./vmlinux.bin",
   11         "boot_args": "console=ttyS0 reboot=k panic=1 pci=off"
   12     }'
   13 
   14 # Set the root filesystem
   15 curl --unix-socket $API_SOCK -i \
   16     -X PUT 'http://localhost/drives/rootfs' \
   17     -H 'Accept: application/json' \
   18     -H 'Content-Type: application/json' \
   19     -d '{
   20         "drive_id": "rootfs",
   21         "path_on_host": "./rootfs.ext4",
   22         "is_root_device": true,
   23         "is_read_only": false
   24     }'
   25 
   26 # Start the VM instance!
   27 curl --unix-socket $API_SOCK -i \
   28     -X PUT 'http://localhost/actions' \
   29     -H 'Accept: application/json' \
   30     -H 'Content-Type: application/json' \
   31     -d '{
   32         "action_type": "InstanceStart"
   33     }'


  At this point, the terminal running the firecracker process will show the boot logs from the kernel, and
  you'll be dropped into the Alpine Linux login prompt, running inside the MicroVM!


  Critical Consideration: The init Process

  You might run into a "Kernel panic - not syncing: No init found" error.


   * Why? Docker containers are often designed to run a single application and don't include a full init
     system like systemd or OpenRC that a traditional OS needs to boot. The kernel boots, looks for /sbin/init
      to run as the first process, can't find it, and panics.
   * Solution: For simple cases, you can tell the kernel to use a shell as the init process. Modify the
     boot_args in your kernel configuration:

   1     "boot_args": "console=ttyS0 reboot=k panic=1 pci=off init=/bin/sh"

      This will boot the kernel and immediately give you a shell prompt, bypassing the need for a full init
  system.


  Summary: Why is this complex?


  This manual process highlights the value provided by platforms like E2B, Weaveworks Ignite, or Kata
  Containers. They automate this entire workflow:
   1. Extracting the container filesystem.
   2. Creating a valid rootfs block device.
   3. Managing and providing a compatible kernel.
   4. Configuring and launching the MicroVM.
   5. Setting up networking and other peripherals.


  You get the developer-friendly experience of Dockerfile with the strong security of Firecracker MicroVMs,
  without needing to perform these low-level steps manually.
