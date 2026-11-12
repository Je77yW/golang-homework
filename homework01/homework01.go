package homework01

// 1. 只出现一次的数字
// 给定一个非空整数数组，除了某个元素只出现一次以外，其余每个元素均出现两次。找出那个只出现了一次的元素。
func SingleNumber(nums []int) int {
	// TODO: implement
	help := make(map[int]int)
	for _, value := range nums {
		help[value]++
	}
	for key, value := range help {
		if value == 1 {
			return key
		}
	}
	return 0
}

// func SingleNumber(nums []int) int {
// 	ans := 0
// 	for _, n := range nums {
// 		ans ^= n
// 	}
// 	return ans
// }

// 2. 回文数
// 判断一个整数是否是回文数
func IsPalindrome(x int) bool {
	// TODO: implement
	if x < 0 {
		return false
	}
	var help []int
	for x != 0 {
		help = append(help, x%10)
		x /= 10
	}
	var length = len(help)
	for index, value := range help {
		if value != help[length-1-index] {
			return false
		}
	}
	return true
}

// func IsPalindrome(x int) bool {
// 	if x < 0 || (x%10 == 0 && x != 0) {
// 		return false
// 	}
// 	rev := 0
// 	for x > rev {
// 		rev = rev*10 + x%10
// 		x /= 10
// 	}
// 	return x == rev || x == rev/10 // 奇数位时多出中间那一位
// }

// 3. 有效的括号
// 给定一个只包括 '(', ')', '{', '}', '[', ']' 的字符串，判断字符串是否有效
// func IsValid(s string) bool {
// 	// TODO: implement
// 	var help []rune = make([]rune, 32)
// 	var i = 0
// 	for _, char := range s {
// 		switch char {
// 		case ')':
// 			if i == 0 || help[i-1] != '(' {
// 				return false
// 			} else {
// 				i--
// 			}
// 		case '}':
// 			if i == 0 || help[i-1] != '{' {
// 				return false
// 			} else {
// 				i--
// 			}
// 		case ']':
// 			if i == 0 || help[i-1] != '[' {
// 				return false
// 			} else {
// 				i--
// 			}
// 		default:
// 			// i 超过 32 再用 append
// 			help[i] = char
// 			i++
// 		}
// 	}
// 	if i > 0 {
// 		return false
// 	}
// 	return true
// }

func IsValid(s string) bool {
	pairs := map[byte]byte{')': '(', '}': '{', ']': '['}
	stack := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if open, ok := pairs[c]; ok {
			if len(stack) == 0 || stack[len(stack)-1] != open {
				return false
			}
			stack = stack[:len(stack)-1]
		} else {
			stack = append(stack, c)
		}
	}
	return len(stack) == 0
}

// 4. 最长公共前缀
// 查找字符串数组中的最长公共前缀
func LongestCommonPrefix(strs []string) string {
	// TODO: implement
	if len(strs) == 0 {
		return ""
	}
	for index := 0; index < len(strs[0]); index++ {
		char := strs[0][index]
		for i := 1; i < len(strs); i++ {
			if index >= len(strs[i]) || strs[i][index] != char {
				return strs[0][:index]
			}
		}
	}
	return strs[0]
}

// 5. 加一
// 给定一个由整数组成的非空数组所表示的非负整数，在该数的基础上加一
func PlusOne(digits []int) []int {
	// TODO: implement
	for i := len(digits) - 1; i >= 0; i-- {
		if digits[i] < 9 {
			digits[i]++
			return digits
		}
		digits[i] = 0
	}
	return append([]int{1}, digits...)
}

// 6. 删除有序数组中的重复项
// 给你一个有序数组 nums ，请你原地删除重复出现的元素，使每个元素只出现一次，返回删除后数组的新长度。
// 不要使用额外的数组空间，你必须在原地修改输入数组并在使用 O(1) 额外空间的条件下完成。
func RemoveDuplicates(nums []int) int {
	// TODO: implement
	if len(nums) == 0 {
		return 0
	}
	slow := 0
	for fast := 1; fast < len(nums); fast++ {
		if nums[fast] != nums[slow] {
			slow++
			nums[slow] = nums[fast]
		}
	}
	return slow + 1
}

// 7. 合并区间
// 以数组 intervals 表示若干个区间的集合，其中单个区间为 intervals[i] = [starti, endi] 。
// 请你合并所有重叠的区间，并返回一个不重叠的区间数组，该数组需恰好覆盖输入中的所有区间。
func Merge(intervals [][]int) [][]int {
	// TODO: implement
	// 要先按 starti 排序
	if len(intervals) <= 1 {
		return intervals
	}
	var result [][]int
	result = append(result, intervals[0])
	for i := 1; i < len(intervals); i++ {
		if intervals[i][0] <= result[len(result)-1][1] && intervals[i][1] > result[len(result)-1][1] {
			result[len(result)-1][1] = intervals[i][1]
		} else {
			result = append(result, intervals[i])
		}
	}
	return result
}

// 8. 两数之和
// 给定一个整数数组 nums 和一个目标值 target，请你在该数组中找出和为目标值的那两个整数
func TwoSum(nums []int, target int) []int {
	// TODO: implement
	for i := 0; i < len(nums)-1; i++ {
		for j := i + 1; j < len(nums); j++ {
			if nums[i]+nums[j] == target {
				return []int{i, j}
			}
		}
	}
	return nil
}

// func TwoSum(nums []int, target int) []int {
// 	seen := make(map[int]int, len(nums))
// 	for i, n := range nums {
// 		if j, ok := seen[target-n]; ok {
// 			return []int{j, i}
// 		}
// 		seen[n] = i
// 	}
// 	return nil
// }
