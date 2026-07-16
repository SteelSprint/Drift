package main

import (
	"fmt"
	"strings"
)

// D! id=reverse_func range-start
func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
// D! id=reverse_func range-end

// D! id=palindrome_func range-start
func isPalindrome(s string) bool {
	s = strings.ToLower(s)
	return s == reverse(s)
}
// D! id=palindrome_func range-end

// D! id=wordcount_func range-start
func wordCount(s string) int {
	return len(strings.Fields(s))
}
// D! id=wordcount_func range-end

func main() {
	fmt.Println(reverse("hello"))
	fmt.Println(isPalindrome("racecar"))
	fmt.Println(wordCount("the quick brown fox"))
}
