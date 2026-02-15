# Tool Go: gRPC-like Client qua WebSocket với GUI (tương tự Postman gRPC)

## Mô tả chung

Xây dựng một ứng dụng desktop bằng **Go** cho phép người dùng gọi các protobuf message qua WebSocket (theo cách hiện tại source đang implement), kèm giao diện GUI để cải thiện trải nghiệm người dùng. Giao diện tham khảo trực tiếp **Postman → tab gRPC Request** để người dùng quen thuộc.

## Yêu cầu GUI

- UI phải rõ ràng, dễ hiểu và đẹp mắt.
- Tất cả các hành động phải được log chi tiết vào một section/console riêng.

## Các thành phần bắt buộc trong giao diện

### 1. Chọn thư mục chứa file .proto

- Một dialog chọn folder.
- Checkbox **“Use default path”** → Khi tick: disable dialog chọn folder. → Default path: Protobuf/
- Handle lỗi nếu folder không tồn tại hoặc không đọc được.

### 2. Dropdown 1 – Chọn file .proto

- Scan **recursive** toàn bộ thư mục (kể cả subfolder) để tìm tất cả file .proto, không bỏ sót file nào.
- Hiển thị danh sách file (ví dụ: Auth.proto, Core.proto, …).

### 3. Dropdown 2 – Chọn message

- Khi chọn file ở dropdown 1 → tự động parse file .proto và load tất cả message vào dropdown 2.
- Hiển thị tên message (ví dụ: AuthRequest, LoginResponse, …).
- Khi chọn message → render form nhập liệu tương ứng (giống Postman gRPC).

### 4. Nút Connect

- Kết nối tới WebSocket.

### 5. Nút Send

- Gửi protobuf message đã điền qua WebSocket.

### 6. Section App Log / Console

- Hiển thị toàn bộ log của ứng dụng.
- Mọi hành động phải được log đầy đủ và chi tiết:
    - Connect / Disconnect
    - Send success / fail
    - Lỗi parse proto
    - Lỗi kết nối
    - Lỗi scan folder, v.v.

## Xử lý WebSocket port

- Đọc từ Windows Registry: Computer\HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\OPSWAT\MD4M Key: ws_port
- Nếu không tìm thấy hoặc lỗi → dùng **default port: 54675**

## Yêu cầu kỹ thuật

- Viết hoàn toàn bằng **Go**.
- Tối ưu performance khi scan/parse nhiều file proto → sử dụng **goroutine + semaphore** để xử lý song song.
- Ứng dụng phải ổn định, xử lý lỗi tốt ở mọi bước.