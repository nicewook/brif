package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

func randomSleep() {
	rand.Seed(time.Now().UnixNano())
	num := rand.Intn(4) + 1 // num is now a number between 1 and 4
	sleepTime := time.Duration(num) * time.Second

	log.Printf("Sleeping for %d seconds...\n", num)
	time.Sleep(sleepTime)
}

/*
Original from the lecture.
The user is asking you to summarize a book. Because the book is too long you are being asked to summarize one
chunk at a time. If a chunk contains a section surrounded by three square brackets, such as

	[[[ some text ]]]

then the content enclosed is itself a GPT-generated summary of another larger chunk. Weigh such summaries with
greater significance than ordinary text they represent the entire passage that they summarize.
In your summary, make no mention of the "chunks" or "passages" used in dividing the text for summarization.
Strive to make your summary as detailed as possible while remaining under a %d token limit.
*/
func promptMessage(text string, targetTokenSize int) []openai.ChatCompletionMessage {
	// refined by GPT-4
	content := fmt.Sprintf(`
The user is requesting a book summary. Due to the extensive length of the book, you're required to summarize it 
one chunk at a time. If a chunk includes a section encapsulated by three square brackets, such as 
    [[[ some text ]]]
, it signifies a GPT-generated summary of a larger chunk. Assign greater importance to these encapsulated summaries, 
as they represent entire passages that they condense.

While drafting your summary, avoid mentioning the 'chunks' or 'passages' that serve as divisions for the summarization process. 
Aim to make your summary as comprehensive as possible, ensuring it remains within a limit of %d tokens.
`, targetTokenSize)
	content = strings.TrimSpace(content)

	systemMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: content,
	}
	userMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: fmt.Sprintf("Summarize the following: %s", text),
	}
	return []openai.ChatCompletionMessage{systemMessage, userMessage}

}

// gptSummarize receive text to summarize and target token size
// then, get response
func gptSummarize(text string, targetTokenSize int) (*openai.ChatCompletionResponse, error) {
	// try 3 times
	for i := 0; i < 3; i++ {
		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:     "gpt-3.5-turbo-0613",
				Messages:  promptMessage(text, targetTokenSize),
				MaxTokens: targetTokenSize,
			},
		)
		if err != nil {
			log.Printf("request %d failed: %v", i, err)
			if i != 2 {
				randomSleep()
			}
			continue
		}
		return &resp, nil

	}
	return nil, errors.New("request 3 times and failed to get GPT summarise")
}

// https://github.com/openai/openai-cookbook/blob/main/examples/How_to_count_tokens_with_tiktoken.ipynb
// func numTokensFromMessages()
