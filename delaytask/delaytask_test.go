package delaytask

import (
	"fmt"
	"testing"
	"time"
)


func TestAddDelayTask(t *testing.T) {
	dt := NewDelayTask()
	dt.Start()
	defer dt.Stop()
	testTask := NewTestTask()
	delayTask.AddRegister(testTask)
	time.Sleep(5 * time.Second)
}

func TestRemoveDelayTask(t *testing.T) {
	dt := NewDelayTask()
	dt.Start()
	defer dt.Stop()
	testTask := NewTestTask()
	testTask.Id = "-1"
	testTask.Delay = 5 * time.Second
	testTask.Options = "options -1"
	AddTask(testTask)
	fmt.Println("task count", Count())
	RemoveTask(-1)
	time.Sleep(1 * time.Second)
	fmt.Println("task count", Count())
}


type TestTask struct {
	Task
}

func (t TestTask) Register() []Task {
	var tasks []Task
	for i := 0; i < 1000; i++ {
		tasks = append(tasks, Task{
			Id:      i,
			Delay:   time.Duration(i) * time.Second,
			Options: fmt.Sprintf("options:%d", i),
		})
	}
	return tasks
}

func (t TestTask) Run(data interface{})  {
	fmt.Println("run test task", data)
}

func NewTestTask() *TestTask {
	return &TestTask{}
}