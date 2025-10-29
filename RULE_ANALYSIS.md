# Code Quality Checker - 규칙 분석 및 검사 코드 유형

본 문서는 `rules.yaml`에 정의된 모든 규칙들과 각 규칙이 검사하는 코드 유형을 상세히 설명합니다.

## 📋 목차

- [Java 규칙](#java-규칙)
- [JavaScript 규칙](#javascript-규칙)
- [HTML 규칙](#html-규칙)
- [CSS 규칙](#css-규칙)
- [구현 상태 요약](#구현-상태-요약)

---

## Java 규칙

### 1. `java-transactional-missing` - @Transactional 어노테이션 누락
**심각도**: High | **카테고리**: transaction

**검사 대상**:
- `@Service` 어노테이션이 있는 클래스의 데이터 변경 메소드
- **복잡한 트랜잭션이 필요한 경우만** 검사 (2024년 개선됨)

**트랜잭션이 필요한 경우**:
1. **여러 Repository 호출** (2개 이상)
2. **조건부 데이터 변경** (if문과 함께하는 데이터 작업)
3. **복합 데이터 작업** (생성/수정/삭제 조합)
4. **외부 시스템 연동과 DB 작업** 동시 수행

**문제가 되는 코드**:
```java
@Service
public class UserService {
    // ❌ 문제: 여러 테이블 작업에 @Transactional 누락
    public void createUserWithOrder(User user, Order order) {
        userRepository.save(user);      // 1번째 Repository
        orderRepository.save(order);    // 2번째 Repository - 트랜잭션 필요!
    }
    
    // ❌ 문제: 조건부 데이터 변경에 @Transactional 누락
    public void updateUserStatus(Long userId, String status) {
        User user = userRepository.findById(userId);
        if ("ACTIVE".equals(status)) {
            user.setStatus(status);
            userRepository.save(user);  // 조건부 데이터 변경 - 트랜잭션 필요!
        }
    }
    
    // ❌ 문제: 복합 작업에 @Transactional 누락
    public void processUserOrder(Long userId, Order newOrder) {
        orderRepository.deleteByUserId(userId);  // 삭제
        orderRepository.save(newOrder);          // 생성 - 복합 작업!
        userRepository.save(user);               // 수정
    }
    
    // ❌ 문제: 외부 시스템 연동과 DB 작업에 @Transactional 누락
    public void createUserAndNotify(User user) {
        userRepository.save(user);               // DB 작업
        emailService.sendWelcomeEmail(user);     // 외부 시스템 - 트랜잭션 필요!
    }
}
```

**올바른 코드**:
```java
@Service
public class UserService {
    // ✅ 올바름: 복잡한 작업에만 @Transactional 적용
    @Transactional
    public void createUserWithOrder(User user, Order order) {
        userRepository.save(user);
        orderRepository.save(order);
    }
    
    @Transactional
    public void updateUserStatus(Long userId, String status) {
        User user = userRepository.findById(userId);
        if ("ACTIVE".equals(status)) {
            user.setStatus(status);
            userRepository.save(user);
        }
    }
    
    @Transactional
    public void processUserOrder(Long userId, Order newOrder) {
        orderRepository.deleteByUserId(userId);
        orderRepository.save(newOrder);
        userRepository.save(user);
    }
    
    @Transactional
    public void createUserAndNotify(User user) {
        userRepository.save(user);
        emailService.sendWelcomeEmail(user);
    }
    
    // ✅ 올바름: 단순한 단일 작업은 @Transactional 불필요 (경고 없음)
    public void saveUser(User user) {
        userRepository.save(user);  // 단일 Repository 호출만
    }
    
    // ✅ 올바름: 단순 조회는 @Transactional 불필요 (경고 없음)
    public User getUser(Long id) {
        return userRepository.findById(id);
    }
}
```

**검사 메시지 예시**:
- `메소드 'createUserWithOrder'에 @Transactional이 필요합니다: 여러 테이블 작업(2개 Repository 호출)`
- `메소드 'updateUserStatus'에 @Transactional이 필요합니다: 조건부 데이터 변경 로직`
- `메소드 'processUserOrder'에 @Transactional이 필요합니다: 복합 데이터 작업(생성/수정/삭제)`
- `메소드 'createUserAndNotify'에 @Transactional이 필요합니다: 외부 시스템 연동과 DB 작업`

### 2. `java-system-out` - System.out.println 사용
**심각도**: Medium | **카테고리**: logging

**검사 대상**:
- `System.out.print()` 패턴
- `System.out.println()` 패턴

**문제가 되는 코드**:
```java
public class UserController {
    public void processUser() {
        // ❌ 문제: System.out 사용
        System.out.println("Processing user...");
        System.out.print("Debug info: ");
    }
}
```

**올바른 코드**:
```java
public class UserController {
    private static final Logger logger = LoggerFactory.getLogger(UserController.class);
    
    public void processUser() {
        // ✅ 올바름: Logger 사용
        logger.info("Processing user...");
        logger.debug("Debug info: ");
    }
}
```

### 3. `java-layer-architecture` - 레이어 아키텍처 위반
**심각도**: High | **카테고리**: architecture

**검사 대상**:
- Controller 클래스에서 DAO/Repository/Mapper 직접 의존

**문제가 되는 코드**:
```java
@RestController
public class UserController {
    // ❌ 문제: Controller에서 DAO 직접 접근
    @Autowired
    private UserDAO userDAO;
    
    @Autowired
    private UserRepository userRepository;
    
    @GetMapping("/users")
    public List<User> getUsers() {
        return userDAO.findAll(); // 레이어 아키텍처 위반
    }
}
```

**올바른 코드**:
```java
@RestController
public class UserController {
    // ✅ 올바름: Service 레이어를 통한 접근
    @Autowired
    private UserService userService;
    
    @GetMapping("/users")
    public List<User> getUsers() {
        return userService.getAllUsers();
    }
}
```

### 4. `java-magic-number` - 매직 넘버 사용
**심각도**: Medium | **카테고리**: maintainability

**검사 대상**:
- 100 이상의 정수 리터럴
- 소수점이 있는 숫자 리터럴
- 제외 대상: 0, 1, 2, 10, 100, 1000

**문제가 되는 코드**:
```java
public class OrderService {
    public void processOrder() {
        // ❌ 문제: 매직 넘버 사용
        if (order.getAmount() > 150) {
            discount = order.getAmount() * 0.15;
        }
        
        for (int i = 0; i < 365; i++) {
            // 365가 무엇을 의미하는지 불분명
        }
    }
}
```

**올바른 코드**:
```java
public class OrderService {
    private static final int DISCOUNT_THRESHOLD = 150;
    private static final double DISCOUNT_RATE = 0.15;
    private static final int DAYS_IN_YEAR = 365;
    
    public void processOrder() {
        // ✅ 올바름: 의미있는 상수 사용
        if (order.getAmount() > DISCOUNT_THRESHOLD) {
            discount = order.getAmount() * DISCOUNT_RATE;
        }
        
        for (int i = 0; i < DAYS_IN_YEAR; i++) {
            // 의미가 명확함
        }
    }
}
```

### 5. `java-method-length` - 메소드 길이 초과
**심각도**: Medium | **카테고리**: maintainability

**검사 대상**:
- 100라인을 초과하는 메소드 (설정 가능)

**문제가 되는 코드**:
```java
public class OrderProcessor {
    // ❌ 문제: 너무 긴 메소드 (100+ 라인)
    public void processComplexOrder(Order order) {
        // 1. 검증 로직 (20라인)
        if (order == null) { /* 검증 코드 */ }
        
        // 2. 할인 계산 로직 (30라인)
        double discount = 0;
        // 복잡한 할인 계산...
        
        // 3. 재고 확인 로직 (25라인)
        // 재고 확인 코드...
        
        // 4. 결제 처리 로직 (30라인)
        // 결제 처리 코드...
        
        // 총 105라인의 긴 메소드
    }
}
```

**올바른 코드**:
```java
public class OrderProcessor {
    // ✅ 올바름: 작은 단위로 분할
    public void processComplexOrder(Order order) {
        validateOrder(order);
        double discount = calculateDiscount(order);
        checkInventory(order);
        processPayment(order, discount);
    }
    
    private void validateOrder(Order order) {
        // 검증 로직만 포함
    }
    
    private double calculateDiscount(Order order) {
        // 할인 계산 로직만 포함
        return 0;
    }
    
    private void checkInventory(Order order) {
        // 재고 확인 로직만 포함
    }
    
    private void processPayment(Order order, double discount) {
        // 결제 처리 로직만 포함
    }
}
```

### 6. `java-exception-handling` - 예외 처리 누락
**심각도**: High | **카테고리**: reliability

**검사 대상**:
- `printStackTrace()` 사용
- `throw new Exception()` 일반적인 예외 사용
- Controller에서 @ControllerAdvice 누락

**문제가 되는 코드**:
```java
public class UserService {
    public void processUser(User user) {
        try {
            // 위험한 작업
            riskyOperation();
        } catch (Exception e) {
            // ❌ 문제: printStackTrace 사용
            e.printStackTrace();
            
            // ❌ 문제: 일반적인 Exception 타입 사용
            throw new Exception("Something went wrong");
        }
    }
}

// ❌ 문제: @ControllerAdvice 없음
@RestController
public class UserController {
    // Controller 메소드들...
}
```

**올바른 코드**:
```java
public class UserService {
    private static final Logger logger = LoggerFactory.getLogger(UserService.class);
    
    public void processUser(User user) {
        try {
            riskyOperation();
        } catch (Exception e) {
            // ✅ 올바름: Logger 사용
            logger.error("Failed to process user", e);
            
            // ✅ 올바름: 구체적인 예외 타입 사용
            throw new UserProcessingException("Failed to process user: " + user.getId(), e);
        }
    }
}

// ✅ 올바름: 전역 예외 처리기
@ControllerAdvice
public class GlobalExceptionHandler {
    @ExceptionHandler(UserProcessingException.class)
    public ResponseEntity<String> handleUserProcessingException(UserProcessingException e) {
        return ResponseEntity.badRequest().body(e.getMessage());
    }
}
```

### 7. `java-input-validation` - 입력값 검증 누락
**심각도**: High | **카테고리**: security

**검사 대상**:
- BenefitValidation 커스텀 검증 로직 사용
- @RequestBody에 @Valid 어노테이션 누락

**문제가 되는 코드**:
```java
@RestController
public class UserController {
    // ❌ 문제: @Valid 어노테이션 누락
    @PostMapping("/users")
    public ResponseEntity<?> createUser(@RequestBody UserDto userDto) {
        return ResponseEntity.ok(userService.save(userDto));
    }
    
    public void validateUser(UserDto dto) {
        // ❌ 문제: 커스텀 검증 로직 사용
        if (BenefitValidation.isEmpty(dto.getName())) {
            throw new IllegalArgumentException("Name is required");
        }
    }
}
```

**올바른 코드**:
```java
@RestController
public class UserController {
    // ✅ 올바름: @Valid 어노테이션 사용
    @PostMapping("/users")
    public ResponseEntity<?> createUser(@RequestBody @Valid UserDto userDto) {
        return ResponseEntity.ok(userService.save(userDto));
    }
}

// ✅ 올바름: Bean Validation 사용
public class UserDto {
    @NotNull
    @Size(min = 2, max = 50)
    private String name;
    
    @Email
    private String email;
    
    // getters and setters
}
```

### 8. `java-cyclomatic-complexity` - 순환 복잡도 초과
**심각도**: Medium | **카테고리**: maintainability

**검사 대상**:
- 순환 복잡도가 10을 초과하는 메소드
- if, else, while, for, switch, catch, 삼항연산자, &&, || 등을 카운트

**문제가 되는 코드**:
```java
public class OrderProcessor {
    // ❌ 문제: 순환 복잡도 높음 (11+)
    public void processOrder(Order order) {
        if (order != null) {                    // +1
            if (order.getStatus() == PENDING) { // +2
                if (order.getAmount() > 100) {  // +3
                    if (order.getCustomer().isPremium()) { // +4
                        // 프리미엄 고객 처리
                    } else {                    // +5
                        // 일반 고객 처리
                    }
                } else if (order.getAmount() > 50) { // +6
                    // 중간 금액 처리
                } else {                        // +7
                    // 소액 처리
                }
            } else if (order.getStatus() == PROCESSING) { // +8
                // 처리 중 로직
            } else if (order.getStatus() == COMPLETED) {  // +9
                // 완료 로직
            } else {                            // +10
                // 기타 상태 처리
            }
        }
        // 복잡도: 11
    }
}
```

**올바른 코드**:
```java
public class OrderProcessor {
    // ✅ 올바름: 메소드 분할로 복잡도 감소
    public void processOrder(Order order) {
        if (order == null) {
            return;
        }
        
        switch (order.getStatus()) {
            case PENDING:
                processPendingOrder(order);
                break;
            case PROCESSING:
                processInProgressOrder(order);
                break;
            case COMPLETED:
                processCompletedOrder(order);
                break;
            default:
                processUnknownStatus(order);
        }
    }
    
    private void processPendingOrder(Order order) {
        if (order.getAmount() > 100) {
            processLargeOrder(order);
        } else if (order.getAmount() > 50) {
            processMediumOrder(order);
        } else {
            processSmallOrder(order);
        }
    }
    
    private void processLargeOrder(Order order) {
        if (order.getCustomer().isPremium()) {
            processPremiumCustomer(order);
        } else {
            processRegularCustomer(order);
        }
    }
}
```

### 9. `java-duplicate-code` - 중복 코드
**심각도**: Medium | **카테고리**: maintainability

**검사 대상**:
- 3회 이상 반복되는 코드 패턴
- 5라인 이상의 동일한 코드 블록

**문제가 되는 코드**:
```java
public class ReportService {
    public void generateUserReport() {
        // ❌ 문제: 중복 패턴
        responseBody.put("status", "success");
        responseBody.put("timestamp", System.currentTimeMillis());
        responseBody.put("data", userData);
    }
    
    public void generateOrderReport() {
        // ❌ 문제: 동일한 패턴 반복
        responseBody.put("status", "success");
        responseBody.put("timestamp", System.currentTimeMillis());
        responseBody.put("data", orderData);
    }
    
    public void generateProductReport() {
        // ❌ 문제: 동일한 패턴 반복
        responseBody.put("status", "success");
        responseBody.put("timestamp", System.currentTimeMillis());
        responseBody.put("data", productData);
    }
}
```

**올바른 코드**:
```java
public class ReportService {
    // ✅ 올바름: 공통 메소드 추출
    private ApiResponse createSuccessResponse(Object data) {
        return ApiResponse.builder()
            .status("success")
            .timestamp(System.currentTimeMillis())
            .data(data)
            .build();
    }
    
    public ApiResponse generateUserReport() {
        return createSuccessResponse(userData);
    }
    
    public ApiResponse generateOrderReport() {
        return createSuccessResponse(orderData);
    }
    
    public ApiResponse generateProductReport() {
        return createSuccessResponse(productData);
    }
}
```

### 10. `java-coding-conventions` - 코딩 컨벤션 위반
**심각도**: Medium | **카테고리**: style

**검사 대상**:
- @Resource와 @Autowired 혼용
- PascalCase 클래스명 위반
- camelCase 메소드/필드명 위반
- 탭과 스페이스 혼용
- 120자 초과 라인

**문제가 되는 코드**:
```java
// ❌ 문제: 클래스명이 PascalCase가 아님
public class user_service {
    
    // ❌ 문제: @Resource와 @Autowired 혼용
    @Resource
    private UserRepository userRepository;
    
    @Autowired
    private OrderService orderService;
    
    // ❌ 문제: 메소드명이 camelCase가 아님
    public void save_user(User user) {
        // ❌ 문제: 너무 긴 라인 (120자 초과)
        if (user != null && user.getName() != null && user.getEmail() != null && user.getAge() > 0 && user.getAddress() != null) {
            userRepository.save(user);
        }
    }
    
    // ❌ 문제: 필드명이 camelCase가 아님
    private String user_name;
}
```

**올바른 코드**:
```java
// ✅ 올바름: PascalCase 클래스명
public class UserService {
    
    // ✅ 올바름: 일관된 @Autowired 사용
    @Autowired
    private UserRepository userRepository;
    
    @Autowired
    private OrderService orderService;
    
    // ✅ 올바름: camelCase 메소드명
    public void saveUser(User user) {
        // ✅ 올바름: 적절한 라인 길이 및 가독성
        if (isValidUser(user)) {
            userRepository.save(user);
        }
    }
    
    private boolean isValidUser(User user) {
        return user != null 
            && user.getName() != null 
            && user.getEmail() != null 
            && user.getAge() > 0 
            && user.getAddress() != null;
    }
    
    // ✅ 올바름: camelCase 필드명
    private String userName;
}
```

### Spring Framework 전용 규칙들

### 11. `spring-validation-missing` - @Valid 어노테이션 누락
**심각도**: Critical | **카테고리**: security

**검사 대상**:
- `@RequestBody` 다음에 `@Valid`가 없는 패턴

**문제가 되는 코드**:
```java
@RestController
public class UserController {
    // ❌ 문제: @Valid 누락
    @PostMapping("/users")
    public ResponseEntity<?> createUser(@RequestBody UserDto userDto) {
        return userService.createUser(userDto);
    }
}
```

**올바른 코드**:
```java
@RestController
public class UserController {
    // ✅ 올바름: @Valid 적용
    @PostMapping("/users")
    public ResponseEntity<?> createUser(@RequestBody @Valid UserDto userDto) {
        return userService.createUser(userDto);
    }
}
```

### 12. `spring-transactional-private` - private 메소드 @Transactional 사용
**심각도**: High | **카테고리**: reliability

**검사 대상**:
- private 메소드에 @Transactional 어노테이션 사용

**문제가 되는 코드**:
```java
@Service
public class UserService {
    // ❌ 문제: private 메소드는 프록시가 작동하지 않음
    @Transactional
    private void updateUserInternal(User user) {
        userRepository.save(user);
    }
}
```

**올바른 코드**:
```java
@Service
public class UserService {
    // ✅ 올바름: public 메소드에 @Transactional 적용
    @Transactional
    public void updateUser(User user) {
        updateUserInternal(user);
    }
    
    // ✅ 올바름: private 메소드는 @Transactional 없음
    private void updateUserInternal(User user) {
        userRepository.save(user);
    }
}
```

### 13. `spring-transactional-rollback` - @Transactional rollbackFor 누락
**심각도**: Medium | **카테고리**: reliability

**검사 대상**:
- rollbackFor 설정이 없는 @Transactional

**문제가 되는 코드**:
```java
@Service
public class UserService {
    // ❌ 문제: 체크드 예외에 대한 rollbackFor 설정 누락
    @Transactional
    public void processUser(User user) throws Exception {
        if (user.getName() == null) {
            throw new Exception("Name is required"); // 롤백되지 않을 수 있음
        }
        userRepository.save(user);
    }
}
```

**올바른 코드**:
```java
@Service
public class UserService {
    // ✅ 올바름: rollbackFor 설정
    @Transactional(rollbackFor = Exception.class)
    public void processUser(User user) throws Exception {
        if (user.getName() == null) {
            throw new Exception("Name is required");
        }
        userRepository.save(user);
    }
}
```

### 14. `spring-security-missing` - 보안 어노테이션 누락
**심각도**: High | **카테고리**: security

**검사 대상**:
- 민감한 메소드에 보안 어노테이션 누락

**문제가 되는 코드**:
```java
@RestController
public class AdminController {
    // ❌ 문제: 민감한 메소드에 보안 어노테이션 없음
    @DeleteMapping("/users/{id}")
    public ResponseEntity<?> deleteUser(@PathVariable Long id) {
        userService.deleteUser(id);
        return ResponseEntity.ok().build();
    }
}
```

**올바른 코드**:
```java
@RestController
public class AdminController {
    // ✅ 올바름: 보안 어노테이션 적용
    @PreAuthorize("hasRole('ADMIN')")
    @DeleteMapping("/users/{id}")
    public ResponseEntity<?> deleteUser(@PathVariable Long id) {
        userService.deleteUser(id);
        return ResponseEntity.ok().build();
    }
}
```

### 15. `spring-secured-deprecated` - @Secured 대신 @PreAuthorize 권장
**심각도**: Medium | **카테고리**: best-practices

**검사 대상**:
- `@Secured` 어노테이션 사용

**문제가 되는 코드**:
```java
@RestController
public class UserController {
    // ❌ 문제: 레거시 @Secured 사용
    @Secured("ROLE_ADMIN")
    @PutMapping("/users/{id}")
    public ResponseEntity<?> updateUser(@PathVariable Long id, @RequestBody UserDto userDto) {
        return userService.updateUser(id, userDto);
    }
}
```

**올바른 코드**:
```java
@RestController
public class UserController {
    // ✅ 올바름: 더 유연한 @PreAuthorize 사용
    @PreAuthorize("hasRole('ADMIN') or #id == authentication.principal.id")
    @PutMapping("/users/{id}")
    public ResponseEntity<?> updateUser(@PathVariable Long id, @RequestBody UserDto userDto) {
        return userService.updateUser(id, userDto);
    }
}
```

### 16. `spring-field-injection` - 필드 주입 대신 생성자 주입 권장
**심각도**: Medium | **카테고리**: best-practices

**검사 대상**:
- `@Autowired private` 패턴

**문제가 되는 코드**:
```java
@Service
public class UserService {
    // ❌ 문제: 필드 주입 사용
    @Autowired
    private UserRepository userRepository;
    
    @Autowired
    private EmailService emailService;
}
```

**올바른 코드**:
```java
@Service
public class UserService {
    // ✅ 올바름: 생성자 주입 사용
    private final UserRepository userRepository;
    private final EmailService emailService;
    
    public UserService(UserRepository userRepository, EmailService emailService) {
        this.userRepository = userRepository;
        this.emailService = emailService;
    }
    
    // 또는 Lombok 사용
    // @RequiredArgsConstructor
}
```

### 17. `spring-controller-advice-missing` - 전역 예외 처리기 누락
**심각도**: High | **카테고리**: reliability

**검사 대상**:
- Controller 클래스에서 @ControllerAdvice 미사용

**문제가 되는 코드**:
```java
// ❌ 문제: 전역 예외 처리기 없음
@RestController
public class UserController {
    @GetMapping("/users/{id}")
    public User getUser(@PathVariable Long id) {
        // 예외 발생 시 처리되지 않음
        return userService.findById(id);
    }
}
```

**올바른 코드**:
```java
@RestController
public class UserController {
    @GetMapping("/users/{id}")
    public User getUser(@PathVariable Long id) {
        return userService.findById(id);
    }
}

// ✅ 올바름: 전역 예외 처리기 추가
@ControllerAdvice
public class GlobalExceptionHandler {
    @ExceptionHandler(EntityNotFoundException.class)
    public ResponseEntity<String> handleNotFound(EntityNotFoundException e) {
        return ResponseEntity.notFound().build();
    }
    
    @ExceptionHandler(ValidationException.class)
    public ResponseEntity<String> handleValidation(ValidationException e) {
        return ResponseEntity.badRequest().body(e.getMessage());
    }
}
```

---

## JavaScript 규칙

### 1. `js-innerHTML-xss` - innerHTML XSS 취약점
**심각도**: Critical | **카테고리**: security

**검사 대상**:
- `.innerHTML = ` 패턴
- 안전한 패턴 제외 (escapeHtml, textContent 등)

**문제가 되는 코드**:
```javascript
// ❌ 문제: XSS 공격 위험
function displayUserInput(userInput) {
    document.getElementById('content').innerHTML = userInput;
    
    // 사용자가 <script>alert('XSS')</script> 입력 시 실행됨
}

function updateMessage(message) {
    element.innerHTML = '<div>' + message + '</div>';
}
```

**올바른 코드**:
```javascript
// ✅ 올바름: textContent 사용
function displayUserInput(userInput) {
    document.getElementById('content').textContent = userInput;
}

// ✅ 올바름: HTML 이스케이프 함수 사용
function updateMessage(message) {
    const escapedMessage = escapeHtml(message);
    element.innerHTML = '<div>' + escapedMessage + '</div>';
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
```

### 2. `js-memory-leak` - 메모리 누수 위험
**심각도**: High | **카테고리**: performance

**검사 대상**:
- 이벤트 리스너 불균형
- 타이머가 정리되지 않음

**문제가 되는 코드**:
```javascript
// ❌ 문제: 이벤트 리스너 제거하지 않음
function setupEventHandlers() {
    document.addEventListener('click', handleClick);
    window.addEventListener('resize', handleResize);
    // 컴포넌트 제거 시 리스너가 남아있음
}

// ❌ 문제: 타이머 정리하지 않음
function startPolling() {
    setInterval(function() {
        fetchData();
    }, 1000);
    // clearInterval 호출하지 않음
}
```

**올바른 코드**:
```javascript
// ✅ 올바름: 이벤트 리스너 정리
class Component {
    constructor() {
        this.handleClick = this.handleClick.bind(this);
        this.handleResize = this.handleResize.bind(this);
    }
    
    mount() {
        document.addEventListener('click', this.handleClick);
        window.addEventListener('resize', this.handleResize);
    }
    
    unmount() {
        document.removeEventListener('click', this.handleClick);
        window.removeEventListener('resize', this.handleResize);
    }
}

// ✅ 올바름: 타이머 정리
class PollingService {
    startPolling() {
        this.intervalId = setInterval(() => {
            this.fetchData();
        }, 1000);
    }
    
    stopPolling() {
        if (this.intervalId) {
            clearInterval(this.intervalId);
            this.intervalId = null;
        }
    }
}
```

### 3. `js-console-log` - console.log 사용
**심각도**: Low | **카테고리**: logging

**검사 대상**:
- `console.log`, `console.warn`, `console.error`, `console.info`, `console.debug` 패턴

**문제가 되는 코드**:
```javascript
// ❌ 문제: 프로덕션 코드에 console 사용
function processOrder(order) {
    console.log('Processing order:', order);
    console.warn('This is a warning');
    console.error('Error occurred');
    
    // 비즈니스 로직...
}
```

**올바른 코드**:
```javascript
// ✅ 올바름: 적절한 로거 사용
const logger = require('./logger');

function processOrder(order) {
    logger.info('Processing order:', order);
    logger.warn('This is a warning');
    logger.error('Error occurred');
    
    // 비즈니스 로직...
}

// 또는 환경별 조건부 로깅
function processOrder(order) {
    if (process.env.NODE_ENV === 'development') {
        console.log('Processing order:', order);
    }
}
```

### 4. `js-var-usage` - var 키워드 사용
**심각도**: Medium | **카테고리**: best-practices

**검사 대상**:
- `var` 키워드 사용

**문제가 되는 코드**:
```javascript
// ❌ 문제: var 사용 (함수 스코프, 호이스팅 문제)
function processItems() {
    for (var i = 0; i < items.length; i++) {
        var item = items[i];
        // var는 함수 스코프이므로 예상치 못한 동작 가능
    }
    
    // i와 item이 여전히 접근 가능
    console.log(i); // items.length
}

// ❌ 문제: 호이스팅으로 인한 문제
function example() {
    console.log(x); // undefined (not ReferenceError)
    var x = 5;
}
```

**올바른 코드**:
```javascript
// ✅ 올바름: let/const 사용 (블록 스코프)
function processItems() {
    for (let i = 0; i < items.length; i++) {
        const item = items[i];
        // 블록 스코프로 안전함
    }
    
    // i와 item은 접근 불가 (ReferenceError)
}

// ✅ 올바름: 호이스팅 문제 해결
function example() {
    const x = 5;
    console.log(x); // 5
}
```

### 5. `js-function-length` - 함수 길이 초과
**심각도**: Medium | **카테고리**: maintainability

**검사 대상**:
- 100라인을 초과하는 함수 (설정 가능)

**문제가 되는 코드**:
```javascript
// ❌ 문제: 너무 긴 함수 (100+ 라인)
function processComplexData(data) {
    // 1. 데이터 검증 (20라인)
    if (!data) return null;
    // 검증 로직...
    
    // 2. 데이터 변환 (30라인)
    let transformed = {};
    // 변환 로직...
    
    // 3. 비즈니스 로직 (35라인)
    // 복잡한 비즈니스 로직...
    
    // 4. 결과 포맷팅 (20라인)
    // 포맷팅 로직...
    
    return result;
}
```

**올바른 코드**:
```javascript
// ✅ 올바름: 작은 함수들로 분할
function processComplexData(data) {
    const validatedData = validateData(data);
    if (!validatedData) return null;
    
    const transformedData = transformData(validatedData);
    const processedData = applyBusinessLogic(transformedData);
    return formatResult(processedData);
}

function validateData(data) {
    // 검증 로직만 포함
}

function transformData(data) {
    // 변환 로직만 포함
}

function applyBusinessLogic(data) {
    // 비즈니스 로직만 포함
}

function formatResult(data) {
    // 포맷팅 로직만 포함
}
```

### 6. `js-strict-mode` - Strict Mode 미사용
**심각도**: Medium | **카테고리**: best-practices

**검사 대상**:
- `'use strict'` 선언이 없는 함수

**문제가 되는 코드**:
```javascript
// ❌ 문제: strict mode 없음
function calculateTotal(items) {
    total = 0; // 전역 변수 생성 (의도하지 않음)
    for (i = 0; i < items.length; i++) { // 전역 변수 생성
        total += items[i].price;
    }
    return total;
}
```

**올바른 코드**:
```javascript
// ✅ 올바름: strict mode 사용
'use strict';

function calculateTotal(items) {
    let total = 0; // ReferenceError 방지
    for (let i = 0; i < items.length; i++) {
        total += items[i].price;
    }
    return total;
}

// 또는 함수별 strict mode
function calculateTotal(items) {
    'use strict';
    let total = 0;
    for (let i = 0; i < items.length; i++) {
        total += items[i].price;
    }
    return total;
}
```

### 7. `js-global-variables` - 전역 변수 사용
**심각도**: Medium | **카테고리**: best-practices

**검사 대상**:
- 전역 스코프에 변수 선언

**문제가 되는 코드**:
```javascript
// ❌ 문제: 전역 변수 사용
var userData = null;
var isLoggedIn = false;
var currentUser = {};

function login(user) {
    userData = user;
    isLoggedIn = true;
    currentUser = user;
}
```

**올바른 코드**:
```javascript
// ✅ 올바름: 모듈 패턴 사용
const UserManager = (function() {
    let userData = null;
    let isLoggedIn = false;
    let currentUser = {};
    
    return {
        login: function(user) {
            userData = user;
            isLoggedIn = true;
            currentUser = user;
        },
        
        logout: function() {
            userData = null;
            isLoggedIn = false;
            currentUser = {};
        },
        
        getCurrentUser: function() {
            return currentUser;
        }
    };
})();

// 또는 ES6 모듈 사용
class UserManager {
    constructor() {
        this.userData = null;
        this.isLoggedIn = false;
        this.currentUser = {};
    }
    
    login(user) {
        this.userData = user;
        this.isLoggedIn = true;
        this.currentUser = user;
    }
}
```

### 8. `js-callback-hell` - 콜백 지옥
**심각도**: High | **카테고리**: maintainability

**검사 대상**:
- 깊게 중첩된 콜백 함수

**문제가 되는 코드**:
```javascript
// ❌ 문제: 콜백 지옥
function fetchUserData(userId) {
    getUserById(userId, function(user) {
        getAddressById(user.addressId, function(address) {
            getOrdersByUserId(userId, function(orders) {
                processOrders(orders, function(processedOrders) {
                    saveProcessedData(processedOrders, function(result) {
                        console.log('All done:', result);
                    });
                });
            });
        });
    });
}
```

**올바른 코드**:
```javascript
// ✅ 올바름: Promise 사용
async function fetchUserData(userId) {
    try {
        const user = await getUserById(userId);
        const address = await getAddressById(user.addressId);
        const orders = await getOrdersByUserId(userId);
        const processedOrders = await processOrders(orders);
        const result = await saveProcessedData(processedOrders);
        console.log('All done:', result);
    } catch (error) {
        console.error('Error:', error);
    }
}

// 또는 Promise 체이닝
function fetchUserData(userId) {
    return getUserById(userId)
        .then(user => getAddressById(user.addressId))
        .then(address => getOrdersByUserId(userId))
        .then(orders => processOrders(orders))
        .then(processedOrders => saveProcessedData(processedOrders))
        .then(result => console.log('All done:', result))
        .catch(error => console.error('Error:', error));
}
```

### 9. `js-unused-variables` - 사용하지 않는 변수
**심각도**: Low | **카테고리**: cleanup

**검사 대상**:
- 선언되었지만 사용되지 않는 변수

**문제가 되는 코드**:
```javascript
// ❌ 문제: 사용하지 않는 변수들
function processData(input) {
    const unusedVariable = 'not used';
    let anotherUnused = 42;
    const data = input.data;
    
    // unusedVariable과 anotherUnused는 사용되지 않음
    return data.processed;
}
```

**올바른 코드**:
```javascript
// ✅ 올바름: 필요한 변수만 선언
function processData(input) {
    const data = input.data;
    return data.processed;
}

// 또는 실제로 사용
function processData(input) {
    const prefix = 'processed_';
    const timestamp = Date.now();
    const data = input.data;
    
    data.id = prefix + timestamp;
    return data.processed;
}
```

### 10. `js-equality-operators` - 동등 연산자 사용
**심각도**: Medium | **카테고리**: best-practices

**검사 대상**:
- `==` 및 `!=` 연산자 사용 (타입 강제 변환)

**문제가 되는 코드**:
```javascript
// ❌ 문제: 타입 강제 변환으로 예상치 못한 결과
function checkValue(value) {
    if (value == 0) {        // '0', false, null도 true
        return 'zero';
    }
    
    if (value != null) {     // undefined도 true
        return 'not null';
    }
    
    // 예상치 못한 동작들
    console.log(0 == '0');     // true
    console.log(false == '0'); // true
    console.log(null == undefined); // true
}
```

**올바른 코드**:
```javascript
// ✅ 올바름: 엄격한 비교 연산자 사용
function checkValue(value) {
    if (value === 0) {        // 정확히 숫자 0만
        return 'zero';
    }
    
    if (value !== null && value !== undefined) {
        return 'not null';
    }
    
    // 예측 가능한 동작들
    console.log(0 === '0');     // false
    console.log(false === '0'); // false
    console.log(null === undefined); // false
}
```

---

## HTML 규칙

### 1. `html-img-alt` - img 태그 alt 속성 누락
**심각도**: High | **카테고리**: accessibility

**검사 대상**:
- `alt` 속성이 없는 `<img>` 태그

**문제가 되는 코드**:
```html
<!-- ❌ 문제: alt 속성 누락 -->
<img src="logo.png">
<img src="user-avatar.jpg" title="User Avatar">
<img src="chart.png" width="300" height="200">
```

**올바른 코드**:
```html
<!-- ✅ 올바름: alt 속성 제공 -->
<img src="logo.png" alt="Company Logo">
<img src="user-avatar.jpg" alt="User Avatar" title="John Doe">
<img src="chart.png" alt="Sales Chart for Q1 2023" width="300" height="200">

<!-- 장식용 이미지의 경우 빈 alt -->
<img src="decoration.png" alt="">
```

### 2. `html-accessibility` - 웹 접근성 위반
**심각도**: High | **카테고리**: accessibility

**검사 대상**:
- 클릭 가능한 div 요소
- 레이블이 없는 버튼

**문제가 되는 코드**:
```html
<!-- ❌ 문제: 클릭 가능한 div (키보드 접근 불가) -->
<div onclick="handleClick()" class="button-like">Click me</div>

<!-- ❌ 문제: 레이블이 없는 버튼 -->
<button onclick="save()">💾</button>

<!-- ❌ 문제: 의미없는 링크 텍스트 -->
<a href="details.html">여기를 클릭하세요</a>
```

**올바른 코드**:
```html
<!-- ✅ 올바름: 의미있는 버튼 사용 -->
<button onclick="handleClick()" class="button-like">Click me</button>

<!-- ✅ 올바름: 명확한 레이블이 있는 버튼 -->
<button onclick="save()" aria-label="Save document">💾</button>

<!-- ✅ 올바름: 의미있는 링크 텍스트 -->
<a href="details.html">제품 상세 정보 보기</a>

<!-- ✅ 올바름: ARIA 속성 사용 -->
<div role="button" 
     tabindex="0" 
     onclick="handleClick()" 
     onkeydown="handleKeyDown(event)"
     aria-label="Custom action button">
    Click me
</div>
```

### 3. `html-seo` - SEO 최적화 누락
**심각도**: Medium | **카테고리**: seo

**검사 대상**:
- title 태그 누락
- meta description 누락
- h1 태그 문제

**문제가 되는 코드**:
```html
<!DOCTYPE html>
<html>
<head>
    <!-- ❌ 문제: title 태그 누락 -->
    <meta charset="UTF-8">
    <!-- ❌ 문제: meta description 누락 -->
</head>
<body>
    <!-- ❌ 문제: h1 태그 없음 -->
    <h2>Welcome to our site</h2>
    <h3>About us</h3>
    
    <!-- ❌ 문제: 여러 개의 h1 태그 -->
    <h1>Main Title</h1>
    <h1>Another Main Title</h1>
</body>
</html>
```

**올바른 코드**:
```html
<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <!-- ✅ 올바름: title 태그 제공 -->
    <title>우리 회사 - 최고의 서비스를 제공합니다</title>
    
    <!-- ✅ 올바름: meta description 제공 -->
    <meta name="description" content="우리 회사는 고객을 위한 최고의 서비스를 제공하는 전문 기업입니다.">
    
    <!-- ✅ 올바름: 추가 SEO 태그 -->
    <meta name="keywords" content="서비스, 품질, 고객만족">
    <meta property="og:title" content="우리 회사">
    <meta property="og:description" content="최고의 서비스를 제공합니다">
</head>
<body>
    <!-- ✅ 올바름: 하나의 h1 태그 -->
    <h1>우리 회사에 오신 것을 환영합니다</h1>
    <h2>서비스 소개</h2>
    <h3>주요 특징</h3>
</body>
</html>
```

### 4. `html-semantic-markup` - 시맨틱 마크업 미사용
**심각도**: Medium | **카테고리**: accessibility

**검사 대상**:
- div 요소에 header, footer, nav 등의 클래스 사용

**문제가 되는 코드**:
```html
<!-- ❌ 문제: div로 시맨틱 영역 표현 -->
<div class="header">
    <div class="nav">
        <a href="home.html">Home</a>
        <a href="about.html">About</a>
    </div>
</div>

<div class="main">
    <div class="article">
        <h1>Article Title</h1>
        <p>Article content...</p>
    </div>
    
    <div class="section">
        <h2>Section Title</h2>
        <p>Section content...</p>
    </div>
</div>

<div class="footer">
    <p>&copy; 2023 Company</p>
</div>
```

**올바른 코드**:
```html
<!-- ✅ 올바름: HTML5 시맨틱 요소 사용 -->
<header>
    <nav>
        <a href="home.html">Home</a>
        <a href="about.html">About</a>
    </nav>
</header>

<main>
    <article>
        <h1>Article Title</h1>
        <p>Article content...</p>
    </article>
    
    <section>
        <h2>Section Title</h2>
        <p>Section content...</p>
    </section>
</main>

<footer>
    <p>&copy; 2023 Company</p>
</footer>
```

### 5. `html-validation` - HTML 유효성 검사
**심각도**: High | **카테고리**: standards

**검사 대상**:
- 닫히지 않은 태그
- 잘못된 중첩

**문제가 되는 코드**:
```html
<!-- ❌ 문제: 닫히지 않은 태그들 -->
<div>
    <p>Some text
    <span>Another text
</div>

<!-- ❌ 문제: 잘못된 중첩 -->
<p>
    <div>This is wrong nesting</div>
</p>

<!-- ❌ 문제: 블록 요소 안에 인라인 요소 잘못 중첩 -->
<a href="link.html">
    <div>Block inside inline</div>
</a>
```

**올바른 코드**:
```html
<!-- ✅ 올바름: 올바르게 닫힌 태그들 -->
<div>
    <p>Some text</p>
    <span>Another text</span>
</div>

<!-- ✅ 올바름: 올바른 중첩 -->
<div>
    <p>This is correct nesting</p>
</div>

<!-- ✅ 올바름: 올바른 인라인/블록 구조 -->
<div>
    <a href="link.html">Inline inside block</a>
</div>
```

### 6. `html-deprecated-tags` - 폐기된 태그 사용
**심각도**: High | **카테고리**: standards

**검사 대상**:
- `<font>`, `<center>`, `<marquee>`, `<blink>` 태그

**문제가 되는 코드**:
```html
<!-- ❌ 문제: HTML5에서 폐기된 태그들 -->
<font color="red" size="3">Deprecated font tag</font>
<center>Centered content</center>
<marquee>Scrolling text</marquee>
<blink>Blinking text</blink>
```

**올바른 코드**:
```html
<!-- ✅ 올바름: CSS를 사용한 스타일링 -->
<span style="color: red; font-size: 1.2em;">Styled text</span>
<div style="text-align: center;">Centered content</div>

<!-- ✅ 올바름: CSS 애니메이션 사용 -->
<style>
.scrolling {
    animation: scroll 10s linear infinite;
}

.blinking {
    animation: blink 1s step-start infinite;
}

@keyframes scroll {
    from { transform: translateX(100%); }
    to { transform: translateX(-100%); }
}

@keyframes blink {
    50% { opacity: 0; }
}
</style>

<div class="scrolling">Scrolling text</div>
<span class="blinking">Blinking text</span>
```

### 7. `html-inline-styles` - 인라인 스타일 사용
**심각도**: Medium | **카테고리**: maintainability

**검사 대상**:
- `style` 속성 사용

**문제가 되는 코드**:
```html
<!-- ❌ 문제: 인라인 스타일 사용 -->
<div style="color: red; font-size: 14px; margin: 10px;">
    Content with inline styles
</div>

<p style="background-color: yellow; padding: 5px;">
    Another styled element
</p>

<button style="background: blue; color: white; border: none;">
    Click me
</button>
```

**올바른 코드**:
```html
<!-- ✅ 올바름: CSS 클래스 사용 -->
<style>
.highlight-text {
    color: red;
    font-size: 14px;
    margin: 10px;
}

.warning-box {
    background-color: yellow;
    padding: 5px;
}

.primary-button {
    background: blue;
    color: white;
    border: none;
    padding: 8px 16px;
    cursor: pointer;
}
</style>

<div class="highlight-text">
    Content with CSS classes
</div>

<p class="warning-box">
    Another styled element
</p>

<button class="primary-button">
    Click me
</button>
```

### 8. `html-form-labels` - 폼 레이블 누락
**심각도**: High | **카테고리**: accessibility

**검사 대상**:
- label과 연결되지 않은 input 요소

**문제가 되는 코드**:
```html
<!-- ❌ 문제: label이 없는 input들 -->
<form>
    Name: <input type="text" name="name">
    <br>
    Email: <input type="email" name="email">
    <br>
    <input type="password" placeholder="Password">
    <br>
    <input type="submit" value="Submit">
</form>
```

**올바른 코드**:
```html
<!-- ✅ 올바름: 적절한 label 연결 -->
<form>
    <label for="name">Name:</label>
    <input type="text" id="name" name="name">
    <br>
    
    <label for="email">Email:</label>
    <input type="email" id="email" name="email">
    <br>
    
    <label for="password">Password:</label>
    <input type="password" id="password" name="password">
    <br>
    
    <!-- 또는 label로 감싸기 -->
    <label>
        Confirm Password:
        <input type="password" name="confirm_password">
    </label>
    <br>
    
    <input type="submit" value="Submit">
</form>
```

---

## CSS 규칙

### 1. `css-selectors` - CSS 셀렉터 효율성
**심각도**: Medium | **카테고리**: performance

**검사 대상**:
- 과도한 중첩 (4단계 이상)
- 전체 선택자 사용
- 비효율적인 후손 선택자

**문제가 되는 코드**:
```css
/* ❌ 문제: 과도한 중첩 셀렉터 */
.container .content .article .header .title .text {
    color: red;
}

/* ❌ 문제: 전체 선택자 사용 */
* {
    margin: 0;
    padding: 0;
}

div * {
    box-sizing: border-box;
}

/* ❌ 문제: 비효율적인 후손 선택자 */
.sidebar div div div span {
    font-weight: bold;
}
```

**올바른 코드**:
```css
/* ✅ 올바름: 간단하고 효율적인 셀렉터 */
.article-title {
    color: red;
}

/* ✅ 올바름: 필요한 요소만 리셋 */
body, h1, h2, h3, p {
    margin: 0;
    padding: 0;
}

/* ✅ 올바름: 구체적인 클래스 사용 */
.sidebar-highlight {
    font-weight: bold;
}

/* ✅ 올바름: 적절한 중첩 (3단계 이하) */
.header .nav .link {
    text-decoration: none;
}
```

### 2. `css-responsive-design` - 반응형 디자인 미적용
**심각도**: Medium | **카테고리**: responsive

**검사 대상**:
- 고정 너비에 미디어 쿼리 없음
- 과도한 px 단위 사용

**문제가 되는 코드**:
```css
/* ❌ 문제: 고정 너비, 미디어 쿼리 없음 */
.container {
    width: 1200px;
    height: 800px;
    font-size: 16px;
    margin: 20px;
    padding: 30px;
}

.sidebar {
    width: 300px;
    float: left;
}

.content {
    width: 900px;
    float: right;
}
```

**올바른 코드**:
```css
/* ✅ 올바름: 유연한 레이아웃과 미디어 쿼리 */
.container {
    max-width: 1200px;
    width: 100%;
    min-height: 100vh;
    font-size: 1rem;
    margin: 0 auto;
    padding: 2rem;
}

.sidebar {
    width: 25%;
    float: left;
}

.content {
    width: 75%;
    float: right;
}

/* 미디어 쿼리로 반응형 구현 */
@media (max-width: 768px) {
    .container {
        padding: 1rem;
    }
    
    .sidebar,
    .content {
        width: 100%;
        float: none;
    }
}

@media (max-width: 480px) {
    .container {
        font-size: 0.9rem;
        padding: 0.5rem;
    }
}
```

### 3. `css-vendor-prefixes` - 벤더 프리픽스 누락
**심각도**: Medium | **카테고리**: compatibility

**검사 대상**:
- transform, transition, animation 속성에 -webkit- 누락

**문제가 되는 코드**:
```css
/* ❌ 문제: 벤더 프리픽스 누락 */
.button {
    transform: scale(1.1);
    transition: all 0.3s ease;
    animation: fadeIn 1s ease-in-out;
}

.box {
    transform: translateX(100px) rotate(45deg);
    transition: transform 0.5s;
}
```

**올바른 코드**:
```css
/* ✅ 올바름: 벤더 프리픽스 포함 */
.button {
    -webkit-transform: scale(1.1);
    -moz-transform: scale(1.1);
    -ms-transform: scale(1.1);
    transform: scale(1.1);
    
    -webkit-transition: all 0.3s ease;
    -moz-transition: all 0.3s ease;
    -ms-transition: all 0.3s ease;
    transition: all 0.3s ease;
    
    -webkit-animation: fadeIn 1s ease-in-out;
    -moz-animation: fadeIn 1s ease-in-out;
    animation: fadeIn 1s ease-in-out;
}

.box {
    -webkit-transform: translateX(100px) rotate(45deg);
    -moz-transform: translateX(100px) rotate(45deg);
    -ms-transform: translateX(100px) rotate(45deg);
    transform: translateX(100px) rotate(45deg);
    
    -webkit-transition: -webkit-transform 0.5s;
    -moz-transition: -moz-transform 0.5s;
    -ms-transition: -ms-transform 0.5s;
    transition: transform 0.5s;
}

@-webkit-keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
}

@keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
}
```

### 4. `css-unused-styles` - 사용하지 않는 CSS
**심각도**: Low | **카테고리**: cleanup

**검사 대상**:
- HTML에서 사용되지 않는 CSS 규칙

**문제가 되는 코드**:
```css
/* ❌ 문제: HTML에서 사용되지 않는 스타일들 */
.unused-class {
    color: red;
}

.old-layout {
    display: flex;
    justify-content: center;
}

.deprecated-style {
    background: linear-gradient(to right, #ff0000, #00ff00);
}

/* 실제 HTML에는 이런 클래스들이 없음 */
```

**올바른 코드**:
```css
/* ✅ 올바름: 실제 사용되는 스타일만 유지 */
.header {
    background-color: #333;
    color: white;
    padding: 1rem;
}

.nav-link {
    color: white;
    text-decoration: none;
}

.content {
    padding: 2rem;
    max-width: 800px;
    margin: 0 auto;
}

/* 해당하는 HTML 요소들이 실제로 존재함 */
```

### 5. `css-important-overuse` - !important 남용
**심각도**: Medium | **카테고리**: maintainability

**검사 대상**:
- `!important` 사용

**문제가 되는 코드**:
```css
/* ❌ 문제: !important 남용 */
.header {
    background-color: blue !important;
    color: white !important;
    padding: 20px !important;
    margin: 0 !important;
}

.button {
    background: red !important;
    color: white !important;
    border: none !important;
    padding: 10px !important;
}

.text {
    font-size: 16px !important;
    line-height: 1.5 !important;
}
```

**올바른 코드**:
```css
/* ✅ 올바름: 적절한 우선순위와 구체성 사용 */
.header {
    background-color: blue;
    color: white;
    padding: 20px;
    margin: 0;
}

.button {
    background: red;
    color: white;
    border: none;
    padding: 10px;
}

.button.primary {
    background: blue; /* 더 구체적인 셀렉터 사용 */
}

.text {
    font-size: 16px;
    line-height: 1.5;
}

/* !important는 정말 필요한 경우에만 사용 */
.accessibility-hide {
    display: none !important; /* 접근성을 위한 숨김 */
}
```

### 6. `css-font-fallbacks` - 폰트 폴백 누락
**심각도**: Medium | **카테고리**: compatibility

**검사 대상**:
- 폴백 폰트가 없는 font-family

**문제가 되는 코드**:
```css
/* ❌ 문제: 폴백 폰트 없음 */
body {
    font-family: "Noto Sans KR";
}

.heading {
    font-family: "CustomFont";
}

.code {
    font-family: "Source Code Pro";
}
```

**올바른 코드**:
```css
/* ✅ 올바름: 적절한 폴백 폰트 체인 */
body {
    font-family: "Noto Sans KR", "Malgun Gothic", "Apple SD Gothic Neo", sans-serif;
}

.heading {
    font-family: "CustomFont", Georgia, "Times New Roman", serif;
}

.code {
    font-family: "Source Code Pro", "Monaco", "Consolas", monospace;
}

/* 시스템 폰트 활용 */
.system-font {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
}
```

### 7. `css-color-contrast` - 색상 대비 부족
**심각도**: High | **카테고리**: accessibility

**검사 대상**:
- 접근성 기준에 미달하는 색상 대비

**문제가 되는 코드**:
```css
/* ❌ 문제: 낮은 색상 대비 */
.low-contrast {
    color: #999999;           /* 회색 텍스트 */
    background-color: #ffffff; /* 흰색 배경 - 대비율 낮음 */
}

.poor-visibility {
    color: #ffff00;           /* 노란색 텍스트 */
    background-color: #ffffff; /* 흰색 배경 - 거의 보이지 않음 */
}

.insufficient {
    color: #666666;           /* 진회색 텍스트 */
    background-color: #cccccc; /* 연회색 배경 - 대비 부족 */
}
```

**올바른 코드**:
```css
/* ✅ 올바름: 충분한 색상 대비 (4.5:1 이상) */
.good-contrast {
    color: #333333;           /* 진한 회색 텍스트 */
    background-color: #ffffff; /* 흰색 배경 - 대비율 충분 */
}

.high-visibility {
    color: #000000;           /* 검은색 텍스트 */
    background-color: #ffffff; /* 흰색 배경 - 최고 대비 */
}

.accessible-dark {
    color: #ffffff;           /* 흰색 텍스트 */
    background-color: #1a1a1a; /* 진한 배경 - 대비 충분 */
}

.warning-accessible {
    color: #cc6600;           /* 주황색 텍스트 */
    background-color: #fff5e6; /* 연한 주황 배경 - 접근성 준수 */
}

/* 대비율 확인 도구 사용 권장:
   - WebAIM Color Contrast Checker
   - WCAG AA 기준: 4.5:1
   - WCAG AAA 기준: 7:1
*/
```

---

## 구현 상태 요약

### ✅ 완전히 구현된 규칙

**Java:**
- `java-transactional-missing` - @Transactional 어노테이션 누락 검사 (⭐ **2024년 개선**: 복잡한 트랜잭션 필요 시에만 검사)
- `java-system-out` - System.out.println 사용 검사  
- `java-layer-architecture` - 레이어 아키텍처 위반 검사
- `java-magic-number` - 매직 넘버 검사
- `java-method-length` - 메소드 길이 검사 (⭐ **2024년 개선**: 100라인 임계값, 설정 가능)
- `java-cyclomatic-complexity` - 순환 복잡도 검사
- `java-duplicate-code` - 중복 코드 검사
- `java-coding-conventions` - 코딩 컨벤션 검사

**JavaScript:**
- `js-innerHTML-xss` - innerHTML XSS 취약점 검사
- `js-console-log` - console.log 사용 검사
- `js-var-usage` - var 키워드 사용 검사
- `js-function-length` - 함수 길이 검사 (100라인 임계값)
- `js-strict-mode` - Strict Mode 미사용 검사
- `js-equality-operators` - 동등 연산자 사용 검사

**HTML:**
- `html-img-alt` - img 태그 alt 속성 누락 검사
- `html-semantic-markup` - 시맨틱 마크업 미사용 검사
- `html-deprecated-tags` - 폐기된 태그 사용 검사
- `html-inline-styles` - 인라인 스타일 사용 검사

**CSS:**
- `css-vendor-prefixes` - 벤더 프리픽스 누락 검사
- `css-important-overuse` - !important 남용 검사
- `css-font-fallbacks` - 폰트 폴백 누락 검사

### ⚠️ 부분적으로 구현된 규칙

**Java:**
- `java-exception-handling` - 부분 구현 (printStackTrace, throw Exception 검사만)
- `java-input-validation` - 부분 구현 (@Valid 누락, 커스텀 검증 검사만)

**Spring:**
- `spring-validation-missing` - 정규식 기반 구현
- `spring-secured-deprecated` - 정규식 기반 구현  
- `spring-field-injection` - 정규식 기반 구현

### ❌ 미구현 규칙

**Spring:**
- `spring-transactional-private` - 조건만 정의됨
- `spring-transactional-rollback` - 조건만 정의됨
- `spring-security-missing` - 조건만 정의됨
- `spring-controller-advice-missing` - 조건만 정의됨

**JavaScript:**
- `js-memory-leak` - 조건만 정의됨
- `js-global-variables` - 조건만 정의됨
- `js-callback-hell` - 조건만 정의됨
- `js-unused-variables` - 조건만 정의됨

**HTML:**
- `html-accessibility` - 조건만 정의됨
- `html-seo` - 조건만 정의됨
- `html-validation` - 조건만 정의됨
- `html-form-labels` - 조건만 정의됨

**CSS:**
- `css-selectors` - 조건만 정의됨
- `css-responsive-design` - 조건만 정의됨
- `css-unused-styles` - 조건만 정의됨
- `css-color-contrast` - 조건만 정의됨

---

## 결론

이 Code Quality Checker는 **Java 규칙**에 대해서는 상당히 완성도 높은 구현을 제공하고 있으며, 특히 Spring Framework 관련 규칙들도 포함하고 있어 실무에서 유용합니다. 

### 🚀 **2024년 주요 개선사항**

1. **`java-transactional-missing` 규칙 고도화**:
   - ❌ 기존: 모든 데이터 변경 메소드에 무조건 @Transactional 요구
   - ✅ 개선: 복잡한 트랜잭션이 필요한 경우만 검사 (여러 Repository, 조건부 로직, 복합 작업, 외부 연동)
   - 📈 실용성 크게 향상, 개발자 수용도 개선

2. **`java-method-length` 규칙 개선**:
   - ❌ 기존: 50라인 하드코딩 임계값
   - ✅ 개선: 100라인 기본값, 설정 파일에서 조정 가능
   - 📈 업계 표준에 맞춤, 실무 적용성 향상

### 📊 **전체 현황**

**JavaScript, HTML, CSS** 규칙들은 기본적인 검사만 구현되어 있어, 더 완전한 코드 품질 검사를 위해서는 추가 구현이 필요한 상태입니다.

전체적으로 **총 37개 규칙** 중 **약 16개 규칙이 완전히 구현**되어 있으며, 나머지는 부분 구현되거나 조건만 정의된 상태입니다.

### 💡 **권장사항**

- **Java 프로젝트**에서는 즉시 실무 적용 가능
- **Spring Boot 프로젝트**에서 특히 유용한 규칙들 다수 포함
- 개선된 @Transactional 규칙으로 **과도한 경고 없이** 효과적인 코드 품질 관리 가능