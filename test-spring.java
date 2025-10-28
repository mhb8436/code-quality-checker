package com.example.controller;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.security.access.annotation.Secured;

@RestController
@RequestMapping("/api/users")
public class UserController {

    // 문제: 필드 주입 사용
    @Autowired
    private UserService userService;

    // 문제: @RequestBody에 @Valid 어노테이션 누락
    @PostMapping
    public ResponseEntity<?> createUser(@RequestBody UserDto userDto) {
        return ResponseEntity.ok(userService.save(userDto));
    }

    // 문제: 민감한 메소드에 보안 어노테이션 누락
    @DeleteMapping("/{id}")
    public ResponseEntity<?> deleteUser(@PathVariable Long id) {
        userService.delete(id);
        return ResponseEntity.ok().build();
    }

    // 문제: @Secured 사용 (레거시)
    @Secured("ROLE_ADMIN")
    @PutMapping("/{id}")
    public ResponseEntity<?> updateUser(@PathVariable Long id, @RequestBody UserDto userDto) {
        return ResponseEntity.ok(userService.update(id, userDto));
    }

    // 문제: private 메소드에 @Transactional 사용
    @Transactional
    private void updateUserInternal(User user) {
        // 이 트랜잭션은 작동하지 않음
        userService.save(user);
    }

    // 문제: @Transactional에 rollbackFor 설정 누락
    @Transactional
    public void processUser(UserDto userDto) throws Exception {
        // 체크드 예외 발생 시 롤백되지 않을 수 있음
        userService.complexOperation(userDto);
    }
}

@Service
public class UserService {

    @Autowired
    private UserRepository userRepository;

    public User save(UserDto userDto) {
        return userRepository.save(convertToEntity(userDto));
    }

    public void delete(Long id) {
        userRepository.deleteById(id);
    }

    public User update(Long id, UserDto userDto) {
        User user = userRepository.findById(id).orElseThrow();
        // update logic
        return userRepository.save(user);
    }

    public void complexOperation(UserDto userDto) throws Exception {
        // 복잡한 비즈니스 로직
        if (userDto.getName() == null) {
            throw new Exception("Name is required");
        }
    }

    private User convertToEntity(UserDto dto) {
        // conversion logic
        return new User();
    }
}