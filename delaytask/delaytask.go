package delaytask

import (
	"github.com/ouqiang/timewheel"
	"sync"
	"time"
)

type Task struct {
	Id interface{}
	Delay time.Duration
	Options interface{}
}

func (t Task) self() Task {
	return t
}

type Tasker interface {
	self() Task
	Register() []Task
	Run(interface{})
}

type DelayTask struct {
	tw *timewheel.TimeWheel
	count uint64
	lock sync.Mutex
}

var delayTask *DelayTask

func (d *DelayTask) Start() {
	d.tw.Start()
}

func (d *DelayTask) Stop() {
	d.tw.Stop()
}

func (d *DelayTask) addTask(delay time.Duration, taskId, options interface{})  {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.tw.AddTimer(delay, taskId, options)
	d.count += 1
}

func (d *DelayTask) AddRegister(tasker ...Tasker)  {
	for i := range tasker {
		t := tasker[i]
		tasks := t.Register()
		for _, task := range tasks {
			options := task.Options
			d.addTask(task.Delay, task.Id, func() {
				t.Run(options)
			})
		}
	}
}

func (d *DelayTask) RemoveTask(taskId interface{}) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.tw.RemoveTimer(taskId)
	d.count -= 1
}

func NewDelayTask() *DelayTask {
	delayTask = &DelayTask{
		tw: timewheel.New(1 * time.Second, 3600, func(data interface{}) {
			delayTask.lock.Lock()
			defer delayTask.lock.Unlock()
			delayTask.count -= 1
			if callback, ok := data.(func()); ok {
				callback()
			}
		}),
	}
	return delayTask
}

func AddTask(tasker ...Tasker)  {
	for _, t := range tasker {
		task := t.self()
		delayTask.addTask(task.Delay, task.Id, func() {
			t.Run(task.Options)
		})
	}
}

func RemoveTask(taskId interface{})  {
	delayTask.RemoveTask(taskId)
}

func Count() uint64 {
	return delayTask.count
}